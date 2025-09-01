package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// provider defined types satisfy framework

var _ resource.Resource = &RepositoryResource{}
var _ resource.ResourceWithImportState = &RepositoryResource{}


type RepositoryResource struct {
	client *GitHubClient
}


// data model for repo resource
type RepositoryResourceModel struct {
	ID	types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Private types.Bool `tfsdk:"private"`
	HasIssues types.Bool `tfsdk:"has_issues"`
	HasWiki types.Bool `tfsdk:"has_wiki"`
	AutoInit types.Bool `tfsdk:"auto_init"`
	FullName types.String `tfsdk:"full_name"`
	Owner types.String `tfsdk:"owner"`
}

func NewRepositoryResource() resource.Resource {
	return &RepositoryResource{}
}

func (r *RepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *RepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "GitHub repository resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Repository ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Repository name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Repository description",
				Optional:            true,
			},
			"private": schema.BoolAttribute{
				MarkdownDescription: "Whether the repository is private",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"has_issues": schema.BoolAttribute{
				MarkdownDescription: "Whether to enable issues for the repository",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"has_wiki": schema.BoolAttribute{
				MarkdownDescription: "Whether to enable wiki for the repository",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"auto_init": schema.BoolAttribute{
				MarkdownDescription: "Whether to create an initial commit with empty README",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"full_name": schema.StringAttribute{
				MarkdownDescription: "Full name of the repository (owner/name)",
				Computed:            true,
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "Repository owner",
				Computed:            true,
			},
		},
	}
}

func (r *RepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*GitHubClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *GitHubClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *RepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := &CreateRepositoryRequest{
		Name: data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Private: data.Private.ValueBool(),
		HasIssues: data.HasIssues.ValueBool(),
		HasWiki: data.HasWiki.ValueBool(),
		AutoInit: data.AutoInit.ValueBool(),
	}

	repository, err := r.client.CreateRepo(ctx, createRequest)

	if err != nil {
		resp.Diagnostics.AddError("Client error", fmt.Sprintf("Could not create repo, error: %s", err))
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(repository.ID, 10))
	data.Name = types.StringValue(repository.Name)
	data.Description = types.StringValue(repository.Description)
	data.Private = types.BoolValue(repository.Private)
	data.HasIssues = types.BoolValue(repository.HasIssues)
	data.HasWiki = types.BoolValue(repository.HasWiki)
	data.FullName = types.StringValue(repository.FullName)
	data.Owner = types.StringValue(repository.Owner.Login)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}


func (r *RepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update repository via GitHub API
	updateReq := &UpdateRepositoryRequest{
		Description: data.Description.ValueString(),
		Private:     data.Private.ValueBool(),
		HasIssues:   data.HasIssues.ValueBool(),
		HasWiki:     data.HasWiki.ValueBool(),
	}

	repository, err := r.client.UpdateRepo(ctx, data.Owner.ValueString(), data.Name.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update repository, got error: %s", err))
		return
	}

	// Update model with response data
	data.Description = types.StringValue(repository.Description)
	data.Private = types.BoolValue(repository.Private)
	data.HasIssues = types.BoolValue(repository.HasIssues)
	data.HasWiki = types.BoolValue(repository.HasWiki)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete repository via GitHub API
	err := r.client.DeleteRepo(ctx, data.Owner.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete repository, got error: %s", err))
		return
	}
}

func (r *RepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get repository from GitHub API
	repository, err := r.client.GetRepo(ctx, data.Owner.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read repository, got error: %s", err))
		return
	}

	// Update model with latest data
	data.Description = types.StringValue(repository.Description)
	data.Private = types.BoolValue(repository.Private)
	data.HasIssues = types.BoolValue(repository.HasIssues)
	data.HasWiki = types.BoolValue(repository.HasWiki)
	data.FullName = types.StringValue(repository.FullName)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: owner/repository_name
	// For now, we'll use the ID field to store this
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("full_name"), req.ID)...)
}