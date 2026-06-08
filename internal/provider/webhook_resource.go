package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

var webhookEvents = []string{
	"artifact.published", "artifact.deleted",
	"repo.created", "repo.updated", "repo.deleted",
	"proxy.error",
}

// NewWebhookResource is registered in provider.Resources.
func NewWebhookResource() resource.Resource { return &webhookResource{} }

type webhookResource struct {
	client *client.Client
}

type webhookModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	URL    types.String `tfsdk:"url"`
	Secret types.String `tfsdk:"secret"`
	Events types.Set    `tfsdk:"events"`
	Active types.Bool   `tfsdk:"active"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An outbound webhook that POSTs repository/artifact events to a URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true},
			"url":  schema.StringAttribute{Required: true, Description: "Endpoint that receives the POST."},
			"secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "HMAC-SHA256 signing key. Write-only — the API never returns it, so it is kept from config and never refreshed.",
			},
			"events": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Event types to deliver: " + fmt.Sprintf("%v", webhookEvents) + ".",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(webhookEvents...)),
				},
			},
			"active": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Whether the webhook delivers events.",
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

// fromAPI updates everything EXCEPT secret (the API never returns it, so the
// configured value is preserved to avoid spurious drift).
func (m *webhookModel) fromAPI(ctx context.Context, w *client.Webhook) {
	m.ID = types.StringValue(w.ID)
	m.Name = types.StringValue(w.Name)
	m.URL = types.StringValue(w.URL)
	events := w.Events
	if events == nil {
		events = []string{}
	}
	m.Events, _ = types.SetValueFrom(ctx, types.StringType, events)
	m.Active = types.BoolValue(w.Active)
}

func (m *webhookModel) toAPI(ctx context.Context) *client.Webhook {
	var events []string
	_ = m.Events.ElementsAs(ctx, &events, false)
	return &client.Webhook{
		ID:     m.ID.ValueString(),
		Name:   m.Name.ValueString(),
		URL:    m.URL.ValueString(),
		Secret: m.Secret.ValueString(),
		Events: events,
		Active: m.Active.ValueBool(),
	}
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateWebhook(ctx, plan.toAPI(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Create webhook failed", err.Error())
		return
	}
	// The API forces active=true on create; honor an explicit active=false.
	if created.Active != plan.Active.ValueBool() {
		body := plan.toAPI(ctx)
		body.ID = created.ID
		if updated, uerr := r.client.UpdateWebhook(ctx, body); uerr == nil {
			created = updated
		}
	}
	plan.fromAPI(ctx, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	w, err := r.client.GetWebhook(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read webhook failed", err.Error())
		return
	}
	state.fromAPI(ctx, w) // secret preserved
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state webhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	updated, err := r.client.UpdateWebhook(ctx, plan.toAPI(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Update webhook failed", err.Error())
		return
	}
	plan.fromAPI(ctx, updated) // keeps plan.Secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteWebhook(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete webhook failed", err.Error())
	}
}

// ImportState accepts the webhook NAME, resolves it to the API ID via list.
// The secret cannot be imported (write-only) and must be re-supplied in config.
func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListWebhooks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}
	for _, w := range all {
		if w.Name == req.ID {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), w.ID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("webhook %q not found", req.ID))
}
