package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
	"github.com/synology-community/go-synology/pkg/api/core"
)

type NotificationTemplateResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	IsDefault types.Bool   `tfsdk:"is_default"`
	Settings  types.Map    `tfsdk:"settings"`
}

var (
	_ resource.Resource                = &NotificationTemplateResource{}
	_ resource.ResourceWithImportState = &NotificationTemplateResource{}
)

func NewNotificationTemplateResource() resource.Resource {
	return &NotificationTemplateResource{}
}

type NotificationTemplateResource struct {
	client core.Api
}

func expandNotificationTemplateSettings(settings map[string]bool) []core.NotificationTemplateSetting {
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	res := make([]core.NotificationTemplateSetting, 0, len(keys))
	for _, key := range keys {
		res = append(res, core.NotificationTemplateSetting{
			Tag:     key,
			Enabled: settings[key],
		})
	}

	return res
}

func flattenNotificationTemplateSettings(
	settings []core.NotificationTemplateSetting,
) types.Map {
	values := make(map[string]attr.Value, len(settings))
	for _, setting := range settings {
		if setting.Tag == "" {
			continue
		}
		values[setting.Tag] = types.BoolValue(setting.Enabled)
	}

	return types.MapValueMust(types.BoolType, values)
}

func mergeNotificationTemplateSettings(
	settings []core.NotificationTemplateSetting,
	known map[string]bool,
) types.Map {
	values := make(map[string]attr.Value, len(settings)+len(known))
	for key, enabled := range known {
		values[key] = types.BoolValue(enabled)
	}
	for _, setting := range settings {
		if setting.Tag == "" {
			continue
		}
		values[setting.Tag] = types.BoolValue(setting.Enabled)
	}

	return types.MapValueMust(types.BoolType, values)
}

func isNotFoundError(err error) bool {
	var notFoundError api.NotFoundError
	if errors.As(err, &notFoundError) {
		return true
	}

	var apiError api.ApiError
	return errors.As(err, &apiError) && apiError.Code == 404
}

func (r *NotificationTemplateResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data NotificationTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := map[string]bool{}
	resp.Diagnostics.Append(data.Settings.ElementsAs(ctx, &settings, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.NotificationTemplateCreate(
		ctx,
		core.NotificationTemplateCreateRequest{
			TemplateName: data.Name.ValueString(),
			Settings:     expandNotificationTemplateSettings(settings),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create notification template", err.Error())
		return
	}

	template, err := r.client.NotificationTemplateGet(ctx, createResp.TemplateID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created notification template", err.Error())
		return
	}

	flattenedSettings := mergeNotificationTemplateSettings(template.Settings, settings)

	data.ID = types.Int64Value(template.TemplateID)
	data.Name = types.StringValue(template.EffectiveName())
	data.IsDefault = types.BoolValue(template.DefaultTemplate())
	data.Settings = flattenedSettings

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationTemplateResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data NotificationTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.NotificationTemplateGet(ctx, data.ID.ValueInt64())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read notification template", err.Error())
		return
	}

	knownSettings := map[string]bool{}
	resp.Diagnostics.Append(data.Settings.ElementsAs(ctx, &knownSettings, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings := mergeNotificationTemplateSettings(template.Settings, knownSettings)

	data.ID = types.Int64Value(template.TemplateID)
	data.Name = types.StringValue(template.EffectiveName())
	data.IsDefault = types.BoolValue(template.DefaultTemplate())
	data.Settings = settings

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationTemplateResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data NotificationTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.NotificationTemplateGet(ctx, data.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read existing notification template", err.Error())
		return
	}

	if current.DefaultTemplate() {
		resp.Diagnostics.AddError(
			"Default notification templates are read-only",
			"Cannot update a default Synology notification template.",
		)
		return
	}

	settings := map[string]bool{}
	resp.Diagnostics.Append(data.Settings.ElementsAs(ctx, &settings, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err = r.client.NotificationTemplateSet(
		ctx,
		core.NotificationTemplateSetRequest{
			TemplateID:   data.ID.ValueInt64(),
			TemplateName: data.Name.ValueString(),
			Settings:     expandNotificationTemplateSettings(settings),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update notification template",
			fmt.Sprintf(
				"%s (template_id=%d template_name=%q settings=%v)",
				err.Error(),
				data.ID.ValueInt64(),
				data.Name.ValueString(),
				settings,
			),
		)
		return
	}

	updated, err := r.client.NotificationTemplateGet(ctx, data.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated notification template", err.Error())
		return
	}

	flattenedSettings := mergeNotificationTemplateSettings(updated.Settings, settings)

	data.Name = types.StringValue(updated.EffectiveName())
	data.IsDefault = types.BoolValue(updated.DefaultTemplate())
	data.Settings = flattenedSettings

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationTemplateResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data NotificationTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.NotificationTemplateGet(ctx, data.ID.ValueInt64())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read notification template before delete", err.Error())
		return
	}

	if current.DefaultTemplate() {
		resp.Diagnostics.AddError(
			"Default notification templates are read-only",
			"Cannot delete a default Synology notification template.",
		)
		return
	}

	err = r.client.NotificationTemplateDelete(ctx, data.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete notification template", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *NotificationTemplateResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = buildName(req.ProviderTypeName, "notification_template")
}

func (r *NotificationTemplateResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Synology DSM notification templates.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Notification template ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Notification template name.",
				Required:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a Synology default template.",
				Computed:            true,
			},
			"settings": schema.MapAttribute{
				MarkdownDescription: "Template settings map where keys are notification tags and values enable or disable each tag.",
				Required:            true,
				ElementType:         types.BoolType,
			},
		},
	}
}

func (r *NotificationTemplateResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(synology.Api)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected client.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	r.client = client.CoreAPI()
}

func (r *NotificationTemplateResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse ID", err.Error())
		return
	}

	template, err := r.client.NotificationTemplateGet(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find notification template", err.Error())
		return
	}

	settings := flattenNotificationTemplateSettings(template.Settings)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), template.TemplateID)...)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("name"), template.EffectiveName())...,
	)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("is_default"), template.DefaultTemplate())...,
	)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("settings"), settings)...)
}
