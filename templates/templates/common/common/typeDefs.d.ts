export {};

declare global {
  /** Represents a template dependency with CPE identifier for security scanning.
   * This is the single source of truth interface for all template dependencies.
   * Used by both template rendering and security scanning. */
  interface TemplateDependency {
    name: string;
    version: string;
    cpe: string;
    ecosystem: string;
    category: "runtime" | "dev" | "peer";
    condition?: string;
  }

  interface Annotation {
    IsType: (type: string) => boolean;
    IsSameType: (a: Annotation) => boolean;
    Type: () => string;
    IsEqual: (a: Annotation) => boolean;
  }

  interface SecurityAnnotation extends Annotation {
    FieldName: string;

    /**
     * Security scheme type from the OAS Security Scheme object "type" field.
     * Values include "apiKey", "http", "oauth2", and "openIdConnect".
     */
    SecType: string;

    /**
     * Underlying security type. Usage is dependent on SecType:
     * - apiKey: Location of API key from the OAS Security Scheme object "in"
     *           field. Values include "cookie", "header", and "query".
     * - http: HTTP Authentication scheme from the OAS Security Scheme object
     *         "scheme" field, normalized to lowercase. Values include "basic",
     *         "bearer", and "custom".
     * - oauth2: OAuth2 Flow type from the OAuth Flows object field name,
     *           normalized to lowercase snakecase. Values include
     *           "client_credentials" and "password".
     * - openIdConnect: N/A
     */
    SubType: string;

    Option: boolean;
    Scheme: boolean;
    SchemeKey: string;
    SecurityOption: boolean;
    Composite: boolean;
  }

  interface HoistedSecurityField {
    Name: string;
    Index: number;
    Group: number;
  }

  interface HoistedSecurityConfig {
    Equivalent: boolean;
    Fields: HoistedSecurityField[];
  }

  /** GoJa Primitive Pointer Type */
  type Pointer<T extends boolean | string | number> = {
    valueOf: () => T;
  } | null;

  type GojaEnum<T extends string> = {
    valueOf: () => T;
    toString: () => string;
  };

  interface OperationSecurityAnnotation extends Annotation {}

  interface JSONAnnotation extends Annotation {
    FieldName: string;
    Ignore: boolean;
  }

  interface ParamAnnotation extends Annotation {
    ParamType: string;
    Name: string;
    Serialization: string;
    Style: string;
    Explode: boolean;
    FieldType: TypeDef;
    AllowReserved: boolean;

    IsGlobal: boolean;
    OperationsForGlobal: string[];
    HasGlobal: boolean;

    /** Enabled if the parameter is global and should be hidden from an
     *  operation (configurable only at the SDK level), such as when the
     *  x-speakeasy-globals-hidden extension is enabled. */
    Hidden: boolean;

    /** Enabled if the parameter is global and the path or operation marks it as
     *  required. Global parameters are implicitly marked as optional, so this
     *  captures the overriding value requirement. */
    RequiredForOperation: boolean;
  }

  interface RequestWrapperAnnotation extends Annotation {}

  interface RequestAnnotation extends Annotation {
    MediaType: string;
  }

  interface MultipartFormAnnotation extends Annotation {
    Name: string;
    File: boolean;
    Content: boolean;
    JSON: boolean;
    FieldType: TypeDef;
  }

  interface FormAnnotation extends Annotation {
    Name: string;
    JSON: boolean;
    Style: string;
    Explode: boolean;
    FieldType: TypeDef;
  }

  interface EncodingAnnotation extends Annotation {
    MediaType: string;
  }

  interface ResponseAnnotation extends Annotation {
    ResultField: boolean;
  }

  interface NeedsCasingAnnotation extends Annotation {}

  enum AnnotationTypes {
    Json = "json",
    Security = "security",
    OperationSecurity = "opSecurity",
    Param = "param",
    Request = "request",
    RequestWrapper = "requestWrapper",
    MultipartForm = "multipartForm",
    Form = "form",
    Encoding = "encoding",
    Response = "response",
    NeedsCasing = "needsCasing",
  }

  interface SpecificAnnotations {
    json: JSONAnnotation;
    security: SecurityAnnotation;
    opSecurity: OperationSecurityAnnotation;
    param: ParamAnnotation;
    request: RequestAnnotation;
    requestWrapper: RequestWrapperAnnotation;
    multipartForm: MultipartFormAnnotation;
    form: FormAnnotation;
    encoding: EncodingAnnotation;
    response: ResponseAnnotation;
    needsCasing: NeedsCasingAnnotation;
  }

  interface Annotations extends Iterable<Annotation> {
    Has: (type: keyof SpecificAnnotations) => boolean;
    Get: <T extends keyof SpecificAnnotations>(
      type: T,
    ) => SpecificAnnotations[T] | null;
  }

  type Job = {
    ID: string;
    FileName: string;
    Context: any;
  };

  type Model = {
    DocGroup: string;
    Name: string;
    Types: TypeDef[];
    Servers?: Servers;
    OutputLocation: string;
  };

  type DocGroupedType = {
    Type: TypeDef;
    DocGroup: string;
  };

  type DocGroupedOperation = {
    Operation: Operation;
    DocGroup: string;
  };

  type AnyValue = {
    Value: any;
  };

  type FieldDef = {
    GetID: () => string;
    Name: string;
    OriginalName: string;
    Type?: TypeDef;
    Annotations?: Annotations;
    Nullable: boolean;
    Optional: boolean;
    Comments?: CommentDef;
    ErrorMessage: boolean;
    Const?: AnyValue;
    Default?: AnyValue;
    IsAdditionalProperties: boolean;
    IsResponseHeaders: boolean;
    IsResponseMetadata: boolean;

    /**
     * Returns a deep copy of the FieldDef.
     */
    Clone: () => FieldDef;

    /**
     * Finds the equivalent field in the given fields by matching using either
     * the x-speakeasy-match path configuration (when useMatchConfig=true) or
     * sanitized field name comparison. Returns a two-element array:
     * [matchedField, pathSegments].
     */
    FindTerraformEquivalentField?: (
      fields: FieldDefs,
      useMatchConfig: boolean,
    ) => [FieldDef | null, string[] | null];
  };

  type ExternalDocs = {
    Description: string;
    URL: string;
  };

  type SimpleCommentDef = {
    Summary: string;
    Description: string;
  };

  type CommentDef = {
    Summary: string;
    Description: string;
    ExternalDocs?: ExternalDocs;
    Deprecated?: boolean;
    DeprecationMessage?: string;
    DeprecationReplacement?: string;
  };

  type CommentSource = "builtin" | "openapi";

  type Scope =
    | "shared"
    | "operations"
    | "utils"
    | "sdk"
    | "webhooks"
    | "callbacks"
    | "errors";

  type DiscriminatorMapping = {
    Name: string;
    DisplayName: string;
    Type: TypeDef;
  };

  /**
   * Collection of DiscriminatorMapping.
   */
  type DiscriminatorMappings = DiscriminatorMapping[];

  type Discriminator = {
    TypePropertyName: string;
    Mapping: DiscriminatorMappings;
    Inferred: boolean;
  };

  type Enum = {
    Type: TypeDef;
    Values: string[];
    Descriptions?: Record<string, string>;
    Groups?: Record<string, string>;
    Names: string[];
    Open: boolean;
    Format: "enum" | "union" | "";
  };

  type DataType =
    | "string"
    | "integer"
    | "int32"
    | "bigint"
    | "number"
    | "float32"
    | "decimal"
    | "boolean"
    | "date"
    | "date-time"
    | "map"
    | "set"
    | "array"
    | "event-stream"
    | "any"
    | "bytes"
    | "class"
    | "enum"
    | "response"
    | "request"
    | "union"
    | "error"
    | "request-stream"
    | "response-stream"
    | "jsonl";

  type ResponseBodyTarget = {
    Target: FieldDef | TypeDef;
    Path: string;
    Body?: ResponseBodyContent;
    StepIdx: number;
  };

  type InputTarget = {
    Target: FieldDef | TypeDef;
    Path: string;
    Inputs?: FieldDef;
  };

  type OutputTarget = {
    Target: FieldDef | TypeDef;
    Path: string;
    Outputs?: FieldDef;
    StepIdx: number;
  };

  type ExampleReference =
    | {
        Target: ResponseBodyTarget;
        Type: "response.body";
      }
    | {
        Target: InputTarget;
        Type: "inputs";
      }
    | {
        Target: OutputTarget;
        Type: "outputs";
      };

  type ExampleReplacement = {
    Path: string;
    Value?: Example;
  };

  type Example = {
    Name: () => string;
    ToJSON: () => string;
    ToString: () => string;
    Clone: () => Example;
    Description: string;
    Value?: YamlNode;
    Reference?: ExampleReference;
    Replacements: ExampleReplacement[];
  };

  type Validations = {
    MinItems?: number;
    MinLength?: number;
    Minimum?: number;
    MaxItems?: number;
    MaxLength?: number;
    Maximum?: number;
    Pattern?: string;
    UniqueItems?: boolean;
  };

  // Caution: this data structure isn't considered stable: context frame types may be added/removed/adjusted
  //          and as such this should only be used for debugging purposes
  type ContextFrame = {
    Type: string;
    Identifier: string;
    Used: boolean;
    MustUse: boolean;
  };

  type ContextStack = ContextFrame[];

  /**
   * Describes the parsed x-speakeasy-entity extension configuration.
   */
  type Entity = {
    /**
     * All entity names described by the x-speakeasy-entity extension configuration.
     */
    Names: string[];
  };

  /**
   * Describes the parsed x-speakeasy-entity-description extension configuration.
   */
  type EntityDescription = {
    /**
     * Entity description for Terraform action.
     */
    TerraformAction: string;

    /**
     * Entity description for Terraform data resource.
     */
    TerraformDataResource: string;

    /**
     * Entity description for Terraform ephemeral resource.
     */
    TerraformEphemeralResource: string;

    /**
     * Entity description for Terraform managed resource.
     */
    TerraformManagedResource: string;
  };

  /**
   * Describes the parsed x-speakeasy-entity-version extension configuration.
   */
  type EntityVersion = {
    /**
     * Entity version for Terraform managed resource.
     */
    TerraformManagedResource: number;
  };

  /**
   * Tracks source associated type for a hoisted oneOf field.
   * Used to generate UseHoistedValue plan modifier to prevent false drift.
   * Uses PascalCase to match Go struct field names.
   */
  type TerraformHoistedSource = {
    AssociatedTypeName: string;
    FieldName: string;
    /** Path segments from the root to the parent oneOf (empty for root-level oneOf) */
    PathPrefix: string[];
  };

  /**
   * Describes a TypeDef node with a DataType that is not supported for Terraform
   * import state operations. Returned by TypeDef.TerraformInvalidImportTypes().
   */
  type TerraformInvalidImportType = {
    Hierarchy: string;
    TypeName: string;
  };

  /**
   * Describes the parsed x-speakeasy-terraform-custom-default extension configuration.
   */
  type TerraformCustomDefault = {
    /**
     * Go package imports required for the custom default.
     */
    Imports?: string[];

    /**
     * Code rendered into the schema to instantiate the custom default implementation.
     */
    SchemaDefinition: string;
  };

  /**
   * Describes the parsed x-speakeasy-terraform-ignore extension configuration.
   */
  type TerraformIgnore = {
    /**
     * When enabled, the field will be ignored in Terraform data models.
     */
    DataModel: boolean;

    /**
     * When enabled, the field will be ignored in Terraform schema definitions.
     */
    Schema: boolean;
  };

  type TransformConfiguration = {
    Type: "jq";
    Config: string;
  };

  /**
   * Describes a Terraform data resource.
   */
  type TerraformDataResource = {
    /**
     * Description for the data resource. Sourced from
     * x-speakeasy-entity-description configuration, if available.
     */
    Description: string;

    /**
     * SchemaDescription returns the description string for the Terraform schema.
     * If a description is set via x-speakeasy-entity-description, it is returned
     * directly. Otherwise, a default description is generated from the entity name.
     */
    SchemaDescription: () => string;

    /**
     * Mapping of global field names to definitions for all operations. These are
     * added to the entity resource struct type, copied in the Configure() method,
     * made optional in the resource schema, and checked in the resource methods.
     */
    GlobalFields: Record<string, FieldDef>;

    /**
     * Computed Go type name for the data resource data model struct, e.g.
     * "ExampleDataSourceModel".
     */
    GoDataModelTypeName: string;

    /**
     * Computed Go type name for the data resource private data model struct,
     * e.g. "ExampleDataSourcePrivateDataModel".
     */
    GoPrivateDataModelTypeName: string;

    /**
     * Computed Go type name for the data resource struct implementation, e.g.
     * "ExampleDataSource".
     */
    GoTypeName: string;

    /**
     * Whether the entity requires SDK method options in its generated code.
     */
    IncludeSDKMethodOptions: boolean;

    /**
     * Unsanitized name of the data resource type, such as "Thing".
     */
    Name: string;

    /**
     * All operations associated with the data resource.
     */
    Operations: TerraformDataResourceOperations;

    /**
     * Operation security configuration for the data resource. This is set
     * when the underlying operations have operation security defined and the
     * enableOperationSecurity generation configuration flag is enabled. Only
     * a single operation security configuration is supported per data resource,
     * even if multiple operations have differing security configurations. This
     * is intentional to simplify the generated Terraform code and avoid
     * complexity around per-operation security configuration for consumers.
     */
    OperationSecurity: FieldDef | undefined;

    /**
     * Mapping of pagination input field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationInputFields: Record<string, boolean>;

    /**
     * Mapping of pagination output field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationOutputFields: Record<string, boolean>;

    /**
     * Merged Terraform resource schema TypeDef across all operations. Populated
     * by AssembleSchemaTypeDef.
     */
    SchemaTypeDef?: TypeDef;

    /**
     * Deduplicated, sorted list of SDK request conversion methods (To_ prefix)
     * across all operations.
     */
    SDKRequestMethods: TerraformSDKMethod[];

    /**
     * Deduplicated, sorted list of SDK response conversion methods
     * (RefreshFrom_ or RefreshFromArrayOf_ prefix) across operations that
     * refresh the data model.
     */
    SDKResponseMethods: TerraformSDKMethod[];

    /**
     * Server configuration for the data resource. This is defined when
     * the underlying operations have path or operation server URLs defined.
     * Only a single server configuration is supported per data resource,
     * even if multiple operations have differing server URLs defined. This is
     * intentional to simplify the generated Terraform code and avoid complexity
     * around per-operation server configuration for consumers.
     */
    Server: TerraformServer | undefined;

    /**
     * Resolved full Terraform data source type name (e.g.
     * "myprovider_my_data_source"), composed from the resolved provider type
     * name and the snake_case form of Name. Populated by
     * TerraformProvider.AddOrGetDataResource.
     */
    TerraformTypeName: string;
  };

  /**
   * Describes all operations associated with a Terraform data resource.
   */
  type TerraformDataResourceOperations = {
    /**
     * Returns all operations.
     */
    All: () => TerraformOperation[];

    /**
     * Returns operations whose API responses are mapped back into the
     * Terraform data model.
     */
    DataModelRefreshOperations: () => TerraformOperation[];

    /**
     * Ordered read operations. Populated by MergeOperationShards.
     */
    Read: TerraformOperation[];

    /**
     * Merged request shard across all read operations. Populated by
     * MergeOperationShards.
     */
    ReadRequestShard?: TypeDef;

    /**
     * Merged response shard across all read operations. Populated by
     * MergeOperationShards.
     */
    ReadResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all read operations.
     * Populated by MergeOperationShards.
     */
    ReadShard?: TypeDef;
  };

  /**
   * Describes a Terraform action.
   */
  type TerraformAction = {
    /**
     * Description for the action. Sourced from
     * x-speakeasy-entity-description configuration, if available.
     */
    Description: string;

    /**
     * SchemaDescription returns the description string for the Terraform schema.
     * If a description is set via x-speakeasy-entity-description, it is returned
     * directly. Otherwise, a default description is generated from the entity name.
     */
    SchemaDescription: () => string;

    /**
     * Mapping of global field names to definitions for all operations. These are
     * added to the entity resource struct type, copied in the Configure() method,
     * made optional in the resource schema, and checked in the resource methods.
     */
    GlobalFields: Record<string, FieldDef>;

    /**
     * Computed Go type name for the action data model struct, e.g.
     * "ExampleActionModel".
     */
    GoDataModelTypeName: string;

    /**
     * Computed Go type name for the action private data model struct, e.g.
     * "ExampleActionPrivateDataModel".
     */
    GoPrivateDataModelTypeName: string;

    /**
     * Computed Go type name for the action struct implementation, e.g.
     * "ExampleAction".
     */
    GoTypeName: string;

    /**
     * Whether the entity requires SDK method options in its generated code.
     */
    IncludeSDKMethodOptions: boolean;

    /**
     * Unsanitized name of the action type, such as "Thing".
     */
    Name: string;

    /**
     * All operations associated with the action.
     */
    Operations: TerraformActionOperations;

    /**
     * Operation security configuration for the action. This is set when the
     * underlying operations have operation security defined and the
     * enableOperationSecurity generation configuration flag is enabled. Only
     * a single operation security configuration is supported per action,
     * even if multiple operations have differing security configurations.
     * This is intentional to simplify the generated Terraform code and avoid
     * complexity around per-operation security configuration for consumers.
     */
    OperationSecurity: FieldDef | undefined;

    /**
     * Mapping of pagination input field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationInputFields: Record<string, boolean>;

    /**
     * Mapping of pagination output field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationOutputFields: Record<string, boolean>;

    /**
     * Merged Terraform resource schema TypeDef across all operations. Populated
     * by AssembleSchemaTypeDef.
     */
    SchemaTypeDef?: TypeDef;

    /**
     * Deduplicated, sorted list of SDK request conversion methods (To_ prefix)
     * across all operations.
     */
    SDKRequestMethods: TerraformSDKMethod[];

    /**
     * Deduplicated, sorted list of SDK response conversion methods
     * (RefreshFrom_ or RefreshFromArrayOf_ prefix) across operations that
     * refresh the data model.
     */
    SDKResponseMethods: TerraformSDKMethod[];

    /**
     * Server configuration for the action. This is defined when the
     * underlying operations have path or operation server URLs defined.
     * Only a single server configuration is supported per action,
     * even if multiple operations have differing server URLs defined. This is
     * intentional to simplify the generated Terraform code and avoid complexity
     * around per-operation server configuration for consumers.
     */
    Server: TerraformServer | undefined;

    /**
     * Resolved full Terraform action type name (e.g.
     * "myprovider_my_action"), composed from the resolved provider type name
     * and the snake_case form of Name. Populated by
     * TerraformProvider.AddOrGetAction.
     */
    TerraformTypeName: string;
  };

  /**
   * Describes all operations associated with a Terraform action.
   */
  type TerraformActionOperations = {
    /**
     * Returns all operations.
     */
    All: () => TerraformOperation[];

    /**
     * Returns operations whose API responses are mapped back into the
     * Terraform data model.
     */
    DataModelRefreshOperations: () => TerraformOperation[];

    /**
     * Ordered invoke operations. Populated by MergeOperationShards.
     */
    Invoke: TerraformOperation[];

    /**
     * Merged request shard across all invoke operations. Populated by
     * MergeOperationShards.
     */
    InvokeRequestShard?: TypeDef;

    /**
     * Merged response shard across all invoke operations. Populated by
     * MergeOperationShards.
     */
    InvokeResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all invoke operations.
     * Populated by MergeOperationShards.
     */
    InvokeShard?: TypeDef;
  };

  /**
   * Describes a Terraform ephemeral resource.
   */
  type TerraformEphemeralResource = {
    /**
     * Description for the ephemeral resource. Sourced from
     * x-speakeasy-entity-description configuration, if available.
     */
    Description: string;

    /**
     * SchemaDescription returns the description string for the Terraform schema.
     * If a description is set via x-speakeasy-entity-description, it is returned
     * directly. Otherwise, a default description is generated from the entity name.
     */
    SchemaDescription: () => string;

    /**
     * Mapping of global field names to definitions for all operations. These are
     * added to the entity resource struct type, copied in the Configure() method,
     * made optional in the resource schema, and checked in the resource methods.
     */
    GlobalFields: Record<string, FieldDef>;

    /**
     * Computed Go type name for the ephemeral resource data model struct,
     * e.g. "ExampleEphemeralResourceModel".
     */
    GoDataModelTypeName: string;

    /**
     * Computed Go type name for the ephemeral resource private data model
     * struct, e.g. "ExampleEphemeralResourcePrivateDataModel".
     */
    GoPrivateDataModelTypeName: string;

    /**
     * Computed Go type name for the ephemeral resource struct implementation,
     * e.g. "ExampleEphemeralResource".
     */
    GoTypeName: string;

    /**
     * Returns whether the ephemeral resource has close operations defined.
     */
    HasCloseOperations: () => boolean;

    /**
     * Whether the entity requires SDK method options in its generated code.
     */
    IncludeSDKMethodOptions: boolean;

    /**
     * Unsanitized name of the ephemeral resource type, such as "Thing".
     */
    Name: string;

    /**
     * All operations associated with the ephemeral resource.
     */
    Operations: TerraformEphemeralResourceOperations;

    /**
     * Operation security configuration for the ephemeral resource. This is set
     * when the underlying operations have operation security defined and the
     * enableOperationSecurity generation configuration flag is enabled. Only
     * a single operation security configuration is supported per ephemeral
     * resource, even if multiple operations have differing security
     * configurations. This is intentional to simplify the generated Terraform
     * code and avoid complexity around per-operation security configuration for
     * consumers.
     */
    OperationSecurity: FieldDef | undefined;

    /**
     * Mapping of pagination input field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationInputFields: Record<string, boolean>;

    /**
     * Mapping of pagination output field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationOutputFields: Record<string, boolean>;

    /**
     * Merged Terraform resource schema TypeDef across all operations. Populated
     * by AssembleSchemaTypeDef.
     */
    SchemaTypeDef?: TypeDef;

    /**
     * Deduplicated, sorted list of SDK request conversion methods (To_ prefix)
     * across all operations.
     */
    SDKRequestMethods: TerraformSDKMethod[];

    /**
     * Deduplicated, sorted list of SDK response conversion methods
     * (RefreshFrom_ or RefreshFromArrayOf_ prefix) across operations that
     * refresh the data model.
     */
    SDKResponseMethods: TerraformSDKMethod[];

    /**
     * Server configuration for the ephemeral resource. This is defined when
     * the underlying operations have path or operation server URLs defined.
     * Only a single server configuration is supported per ephemeral resource,
     * even if multiple operations have differing server URLs defined. This is
     * intentional to simplify the generated Terraform code and avoid complexity
     * around per-operation server configuration for consumers.
     */
    Server: TerraformServer | undefined;

    /**
     * Resolved full Terraform ephemeral resource type name (e.g.
     * "myprovider_my_ephemeral_resource"), composed from the resolved
     * provider type name and the snake_case form of Name. Populated by
     * TerraformProvider.AddOrGetEphemeralResource.
     */
    TerraformTypeName: string;
  };

  /**
   * Describes all operations associated with a Terraform ephemeral resource.
   */
  type TerraformEphemeralResourceOperations = {
    /**
     * Returns all operations.
     */
    All: () => TerraformOperation[];

    /**
     * Ordered close operations. Populated by MergeOperationShards.
     */
    Close: TerraformOperation[];

    /**
     * Merged request shard across all close operations. Populated by
     * MergeOperationShards.
     */
    CloseRequestShard?: TypeDef;

    /**
     * Merged response shard across all close operations. Populated by
     * MergeOperationShards.
     */
    CloseResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all close operations.
     * Populated by MergeOperationShards.
     */
    CloseShard?: TypeDef;

    /**
     * Returns operations whose API responses are mapped back into the
     * Terraform data model.
     */
    DataModelRefreshOperations: () => TerraformOperation[];

    /**
     * Ordered open operations. Populated by MergeOperationShards.
     */
    Open: TerraformOperation[];

    /**
     * Merged request shard across all open operations. Populated by
     * MergeOperationShards.
     */
    OpenRequestShard?: TypeDef;

    /**
     * Merged response shard across all open operations. Populated by
     * MergeOperationShards.
     */
    OpenResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all open operations.
     * Populated by MergeOperationShards.
     */
    OpenShard?: TypeDef;
  };

  /**
   * Describes a Terraform managed resource.
   */
  type TerraformManagedResource = {
    /**
     * Description for the managed resource. Sourced from
     * x-speakeasy-entity-description configuration, if available.
     */
    Description: string;

    /**
     * SchemaDescription returns the description string for the Terraform schema.
     * If a description is set via x-speakeasy-entity-description, it is returned
     * directly. Otherwise, a default description is generated from the entity name.
     */
    SchemaDescription: () => string;

    /**
     * Mapping of global field names to definitions for all operations. These are
     * added to the entity resource struct type, copied in the Configure() method,
     * made optional in the resource schema, and checked in the resource methods.
     */
    GlobalFields: Record<string, FieldDef>;

    /**
     * Computed Go type name for the managed resource data model struct, e.g.
     * "ExampleResourceModel".
     */
    GoDataModelTypeName: string;

    /**
     * Computed Go type name for the managed resource private data model struct,
     * e.g. "ExampleResourcePrivateDataModel".
     */
    GoPrivateDataModelTypeName: string;

    /**
     * Computed Go type name for the managed resource struct implementation,
     * e.g. "ExampleResource".
     */
    GoTypeName: string;

    /**
     * Whether the entity requires SDK method options in its generated code.
     */
    IncludeSDKMethodOptions: boolean;

    /**
     * List of HTTP status codes that represent the managed resource is not
     * found in the API during read operations. These codes automatically cause
     * the resource to be removed from the Terraform state rather than return an
     * API error.
     *
     * Defaults to 404, which has historically been used by many APIs to
     * represent this situation. 410 could potentially also be considered in the
     * future. Values can be customized via x-speakeasy-entity-missing-codes
     * extension configuration.
     */
    MissingCodes: number[];

    /**
     * Unsanitized name of the managed resource type, such as "Thing".
     */
    Name: string;

    /**
     * All operations associated with the managed resource.
     */
    Operations: TerraformManagedResourceOperations;

    /**
     * Operation security configuration for the managed resource. This is set
     * when the underlying operations have operation security defined and the
     * enableOperationSecurity generation configuration flag is enabled. Only
     * a single operation security configuration is supported per managed
     * resource, even if multiple operations have differing security
     * configurations. This is intentional to simplify the generated Terraform
     * code and avoid complexity around per-operation security configuration for
     * consumers.
     */
    OperationSecurity: FieldDef | undefined;

    /**
     * Mapping of pagination input field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationInputFields: Record<string, boolean>;

    /**
     * Mapping of pagination output field names to an unused boolean value
     * that is replaceable for future updates.
     */
    PaginationOutputFields: Record<string, boolean>;

    /**
     * Terraform schema for the managed resource.
     */
    Schema: TerraformManagedResourceSchema;

    /**
     * Warnings collected during schema assembly. These are non-fatal messages
     * that should be logged after assembly completes.
     */
    SchemaWarnings?: string[];

    /**
     * Fields from the read response shard with x-speakeasy-soft-delete-property.
     * Populated by AssembleSchemaTypeDef.
     */
    ReadSoftDeleteProperties?: FieldDef[];

    /**
     * Import state TypeDef for the managed resource, derived from the read
     * request shard. Contains only required fields (as determined by
     * FieldDef.IsTerraformImportRequired), sorted alphabetically. Used by
     * Terraform import state template generation to determine the import ID
     * shape. Populated by AssembleSchemaTypeDef.
     */
    ImportStateTypeDef?: TypeDef;

    /**
     * Merged Terraform resource schema TypeDef across all operations. Populated
     * by AssembleSchemaTypeDef.
     */
    SchemaTypeDef?: TypeDef;

    /**
     * Deduplicated, sorted list of SDK request conversion methods (To_ prefix)
     * across all operations.
     */
    SDKRequestMethods: TerraformSDKMethod[];

    /**
     * Deduplicated, sorted list of SDK response conversion methods
     * (RefreshFrom_ or RefreshFromArrayOf_ prefix) across operations that
     * refresh the data model.
     */
    SDKResponseMethods: TerraformSDKMethod[];

    /**
     * Server configuration for the managed resource. This is defined when
     * the underlying operations have path or operation server URLs defined.
     * Only a single server configuration is supported per managed resource,
     * even if multiple operations have differing server URLs defined. This is
     * intentional to simplify the generated Terraform code and avoid complexity
     * around per-operation server configuration for consumers.
     */
    Server: TerraformServer | undefined;

    /**
     * Resolved full Terraform managed resource type name (e.g.
     * "myprovider_my_resource"), composed from the resolved provider type
     * name and the snake_case form of Name. Populated by
     * TerraformProvider.AddOrGetManagedResource.
     */
    TerraformTypeName: string;
  };

  /**
   * Describes all operations associated with a Terraform managed resource.
   */
  type TerraformManagedResourceOperations = {
    /**
     * Returns all operations.
     */
    All: () => TerraformOperation[];

    /**
     * Ordered create operations. Populated by MergeOperationShards.
     */
    Create: TerraformOperation[];

    /**
     * Whether the read operation should be invoked after create operations
     * because the read response shard contains entity fields not present in
     * the create response shard. Populated by MergeOperationShards.
     */
    CreateNeedsReadAfter: boolean;

    /**
     * Merged request shard across all create operations. Populated by
     * MergeOperationShards.
     */
    CreateRequestShard?: TypeDef;

    /**
     * Merged response shard across all create operations. Populated by
     * MergeOperationShards.
     */
    CreateResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all create operations.
     * Populated by MergeOperationShards.
     */
    CreateShard?: TypeDef;

    /**
     * Ordered delete operations. Populated by MergeOperationShards.
     */
    Delete: TerraformOperation[];

    /**
     * Merged request shard across all delete operations. Populated by
     * MergeOperationShards.
     */
    DeleteRequestShard?: TypeDef;

    /**
     * Merged response shard across all delete operations. Populated by
     * MergeOperationShards.
     */
    DeleteResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all delete operations.
     * Populated by MergeOperationShards.
     */
    DeleteShard?: TypeDef;

    /**
     * Returns operations whose API responses are mapped back into the
     * Terraform data model.
     */
    DataModelRefreshOperations: () => TerraformOperation[];

    /**
     * Ordered read operations. Populated by MergeOperationShards.
     */
    Read: TerraformOperation[];

    /**
     * Merged request shard across all read operations. Populated by
     * MergeOperationShards.
     */
    ReadRequestShard?: TypeDef;

    /**
     * Merged response shard across all read operations. Populated by
     * MergeOperationShards.
     */
    ReadResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all read operations.
     * Populated by MergeOperationShards.
     */
    ReadShard?: TypeDef;

    /**
     * Ordered update operations. Populated by MergeOperationShards.
     */
    Update: TerraformOperation[];

    /**
     * Whether the read operation should be invoked after update operations
     * because the read response shard contains entity fields not present in
     * the update response shard. Populated by MergeOperationShards.
     */
    UpdateNeedsReadAfter: boolean;

    /**
     * Merged request shard across all update operations. Populated by
     * MergeOperationShards.
     */
    UpdateRequestShard?: TypeDef;

    /**
     * Merged response shard across all update operations. Populated by
     * MergeOperationShards.
     */
    UpdateResponseShard?: TypeDef;

    /**
     * Merged request and response shard across all update operations.
     * Populated by MergeOperationShards.
     */
    UpdateShard?: TypeDef;
  };

  /**
   * Describes the Terraform managed resource schema configuration.
   */
  type TerraformManagedResourceSchema = {
    /**
     * Schema version for the managed resource. This relates to Terraform's
     * managed resource state upgrade mechanism. By default and in most cases,
     * this is never set and left as the default value of 0. Sourced from
     * x-speakeasy-entity-version configuration, if available.
     */
    Version: number;
  };

  /**
   * Represents a TypeDef along the path from an API request or response type
   * to a Terraform entity TypeDef that needs an SDK conversion method.
   */
  type TerraformSDKMethodTarget = {
    /**
     * The target type that needs a conversion method.
     */
    TypeDef: TypeDef;

    /**
     * Returns a TerraformSDKMethod for an SDK-to-model (RefreshFrom_ or
     * RefreshFromArrayOf_) conversion.
     */
    SDKResponseMethod: () => TerraformSDKMethod;
  };

  /**
   * Represents a deduplicated SDK conversion method for a Terraform entity's
   * data model type. Contains the resolved method name and operation context
   * needed for method body rendering.
   */
  type TerraformSDKMethod = {
    /**
     * Fully sanitized method name, e.g. "ToOperationsCreateThingRequest"
     * or "RefreshFromSharedThing".
     */
    MethodName: string;

    /**
     * The first TerraformOperation that contributed this method during
     * deduplication (first-wins). Provides operation context such as patch
     * style, pagination, and entity operation string.
     */
    Operation: TerraformOperation;

    /**
     * Whether the SDK parameter for response methods should be a pointer type.
     */
    Optional: boolean;

    /**
     * The underlying SDK method target containing the TypeDef and metadata.
     */
    Target: TerraformSDKMethodTarget;
  };

  /**
   * Describes a Terraform operation, which is an API operation that has been
   * mapped to a Terraform resource operation. This accounts for configuration,
   * such as the x-speakeasy-entity extension, that modifies the operation
   * request and response data in preparation for merging all operation data
   * into a single Terraform resource schema.
   */
  type TerraformOperation = {
    /**
     * Original API operation. Should always remain untouched.
     */
    APIOperation: Operation;

    /**
     * HTTP status codes that indicate the entity is not found by the API.
     * Set on read and delete operations for managed resources.
     */
    EntityMissingCodes: number[];

    /**
     * Name of the entity this operation is associated with.
     */
    EntityName: string;

    /**
     * String representation from the entity operation configuration in the form
     * of Entity#Op[#Order], such as Thing#create.
     */
    EntityOperation: string;

    /**
     * Whether the operation's API response spec defines a 409 Conflict
     * response code.
     */
    Has409ConflictCode: boolean;

    /**
     * Whether this operation has API-level security defined and the entity's
     * enableOperationSecurity generation config flag is enabled.
     */
    IncludeOperationSecurity: boolean;

    /**
     * Options associated with the entity operation configuration.
     */
    Options: ExtEntityOperationV1Options | undefined;

    /**
     * Operation request shard. Unset if there is no relevant request.
     *
     * A shard represents the portion of the data that is most relevant to the
     * Terraform resource. For example, the x-speakeasy-entity extension
     * is used to indicate which data represents the top level of the Terraform
     * resource schema, potentially bypassing API-defined intermediate
     * properties such as "data" that are undesirable in Terraform
     * configurations. Resource logic is generated to handle the translation
     * between the API-defined schema and the Terraform resource schema.
     */
    RequestShard: TypeDef | undefined;

    /**
     * SDK method for the operation's root request type. Undefined when the
     * operation has no request body or no matching target exists.
     */
    RequestBodySDKMethod: TerraformSDKMethod | undefined;

    /**
     * Response body FieldDef for this operation. Undefined when the response
     * has no body field relevant to the entity or when the operation skips
     * data model refresh. A nil check on this field alone is sufficient to
     * determine whether a data model refresh applies.
     */
    ResponseBodyFieldDef: FieldDef | undefined;

    /**
     * Whether this operation has a Pagination extension and the response
     * body represents a direct entity (not an array extraction). When false,
     * pagination loops and pre-refresh field reinitialization are suppressed.
     */
    SupportsPagination: boolean;

    /**
     * SDK method for the operation's response body type. Undefined when no
     * matching target or ResponseBodyFieldDef exists.
     */
    ResponseBodySDKMethod: TerraformSDKMethod | undefined;

    /**
     * Returns the response SDK method target whose TypeDef matches the
     * given TypeDef. Returns undefined if no match is found.
     */
    FindResponseSDKMethodTarget: (
      td: TypeDef,
    ) => TerraformSDKMethodTarget | undefined;

    /**
     * Terraform schema attribute name used to look up the per-operation
     * server URL from state. Empty string indicates no server URL support.
     */
    ServerAttributeName: string;

    /**
     * Unique 2xx HTTP status codes defined in the operation's API response
     * spec. Codes containing "X" (e.g. "2XX") are included as-is.
     */
    SuccessCodes: string[];

    /**
     * Returns a deep copy of the TerraformOperation.
     */
    Clone: () => TerraformOperation;
  };

  /**
   * Terraform Provider as gathered from various x-speakeasy-entity*
   * extensions across the source document. Only set when generation target
   * is "terraform".
   */
  type TerraformProvider = {
    /**
     * Valid actions sorted by name. Populated by AssembleSchemas.
     *
     * Terraform actions are entities with an invoke lifecycle operation.
     * These are represented in Terraform using "action" configuration blocks.
     */
    Actions: TerraformAction[];

    /**
     * Valid data resources sorted by name. Populated by AssembleSchemas.
     *
     * Terraform data resources are entities with only a Read lifecycle
     * operation. These are represented in Terraform using "data" configuration
     * blocks and colloquially referred to as "data sources".
     */
    DataResources: TerraformDataResource[];

    /**
     * Valid ephemeral resources sorted by name. Populated by AssembleSchemas.
     *
     * Terraform ephemeral resources are entities with an open lifecycle
     * operation. These are represented in Terraform using "ephemeral"
     * configuration blocks.
     */
    EphemeralResources: TerraformEphemeralResource[];

    /**
     * Valid managed resources sorted by name. Populated by AssembleSchemas.
     *
     * Terraform managed resources are entities with Create, Read, Update,
     * and Delete lifecycle operations. These are represented in Terraform
     * using "resource" configuration blocks and colloquially referred to as
     * "resources" due to that implementation detail and existing before other
     * resource types in Terraform.
     */
    ManagedResources: TerraformManagedResource[];

    /**
     * Resolved Terraform provider type name (e.g. "myprovider"). Used as the
     * value of resp.TypeName in the provider Metadata implementation, and as
     * the prefix for all resource, data source, ephemeral resource, and
     * action type names. Sourced from the providerTypeNameOverride
     * generation configuration if set, otherwise from packageName. Must be
     * set before any AddOrGet* method is called so that entity
     * TerraformTypeName fields are composed with the correct prefix.
     */
    TerraformTypeName: string;
  };

  /**
   * Describes a Terraform server configuration.
   */
  type TerraformServer = {
    /**
     * Name of the configurable attribute in the schema.
     */
    AttributeName: string;

    /**
     * Description of the server configuration.
     */
    Description: string;

    /**
     * URL template for the server configuration.
     */
    URL: string;
  };

  /**
   * TypeDef extension configurations.
   */
  type PublicExport = {
    Group: string;
    Name: string;
    /**
     * Which rendering of the target the alias refers to: "model" (default)
     * or "input" (the request-input rendering, e.g. Python's TypedDict
     * companion).
     */
    Representation?: "model" | "input";
  };

  type ResolvedPublicExportTarget = {
    Name: string;
    Target: TypeDef;
    Input?: boolean;
    /**
     * Set for exports registered from x-speakeasy-model-namespace tagging
     * rather than an explicit x-speakeasy-exports declaration. Only rendered
     * when the SDK opts in via the imports.paths.resources configuration.
     */
    Implicit?: boolean;
  };

  type ResolvedPublicExportChild = {
    Name: string;
    Group: string;
    Parts: string[];
  };

  type ResolvedPublicExportGroup = {
    Group: string;
    Parts: string[];
    Exports?: ResolvedPublicExportTarget[];
    Children?: ResolvedPublicExportChild[];
  };

  type ResolvedPublicExports = {
    RootChildren?: ResolvedPublicExportChild[];
    Groups?: ResolvedPublicExportGroup[];
  };

  type TypeDefExtensions = {
    /**
     * All extensions, including those not represented in other fields. Ideally,
     * any TypeDef extensions should be fully validated and typed for templating
     * rather than relying on this catch-all.
     */
    All: Record<string, any>;

    /**
     * Describes x-speakeasy-base64-input-mode extension configuration.
     * Empty when the extension is not applied. Set on request-side string
     * schemas with format:byte or contentEncoding:base64.
     */
    Base64InputMode?: "" | "file";

    /**
     * Describes x-speakeasy-pattern-error-message extension configuration.
     * Empty when the extension is not applied.
     */
    PatternErrorMessage?: string;

    /**
     * Describes x-speakeasy-entity extension configuration.
     */
    Entity?: Entity;

    /**
     * Describes x-speakeasy-entity-description extension configuration.
     */
    EntityDescription?: EntityDescription;

    /**
     * Describes x-speakeasy-entity-version extension configuration.
     */
    EntityVersion?: EntityVersion;

    /**
     * Describes x-speakeasy-example-unset extension configuration.
     */
    ExampleUnset?: boolean;

    /**
     * Describes x-speakeasy-ignore extension configuration.
     */
    Ignore?: boolean;

    /**
     * Describes x-speakeasy-model-namespace extension configuration.
     * When set, the type is placed in a custom namespace/folder for the model output.
     */
    ModelNamespace?: string;

    /**
     * Describes x-speakeasy-match extension configuration.
     */
    MatchConfig?: {
      /**
       * Path to match in the entity (e.g., "id", "object.id").
       * When x-speakeasy-match is specified as a scalar string, this field is populated.
       */
      Path?: string;

      /**
       * Whether to use the prior state value for this parameter in update operations.
       */
      UsePriorState?: boolean;
    };

    /**
     * Describes x-speakeasy-overridable-scopes extension configuration.
     */
    OverridableOAuth2Scopes?: boolean;

    /**
     * Describes x-speakeasy-exports extension configuration.
     * Public export aliases keyed by SDK group path.
     */
    PublicExports?: PublicExport[];

    /**
     * Describes x-speakeasy-pagination extension configuration.
     *
     * NOTE: This is not parsed directly for TypeDef, but instead added to the
     * response TypeDef when configured on the containing operation.
     */
    Pagination?: PaginationConfig;

    /**
     * Describes x-speakeasy-terraform-alias-to extension configuration.
     */
    TerraformAliasTo?: string;

    /**
     * Describes x-speakeasy-terraform-custom-default extension configuration.
     */
    TerraformCustomDefault?: TerraformCustomDefault;

    /**
     * Describes x-speakeasy-response-filter extension configuration.
     */
    ResponseFilter?: boolean;

    /**
     * Describes x-speakeasy-terraform-ignore extension configuration.
     */
    TerraformIgnore?: TerraformIgnore;

    /**
     * Describes x-speakeasy-terraform-write-only extension configuration.
     */
    TerraformWriteOnly?: boolean;

    /**
     * Tracks source associated types for hoisted oneOf fields.
     * When set, generates UseHoistedValue plan modifier to prevent false drift.
     * Uses PascalCase to match Go struct field names.
     */
    TerraformHoistedFrom?: TerraformHoistedSource[];

    /**
     * Describes x-speakeasy-transform-from-api extension configuration.
     */
    TransformFromAPI?: TransformConfiguration;

    /**
     * Describes x-speakeasy-transform-to-api extension configuration.
     */
    TransformToAPI?: TransformConfiguration;
  };

  /**
   * Collection of FieldDef.
   */
  type FieldDefs = FieldDef[];

  /**
   * Collection of TypeDef.
   */
  type TypeDefs = TypeDef[];

  type TypeDef = {
    Name: string;
    OriginalName: string;
    Type: GojaEnum<DataType>;
    ItemType?: TypeDef;
    ContainsNull: boolean;
    Fields: FieldDefs;
    ContextStack: ContextStack;
    AssociatedTypes: TypeDefs;
    Enum?: Enum;
    Scope: Scope;
    IsComponent: boolean;
    Truncated: boolean;
    Comments?: CommentDef;
    Input: boolean;
    Output: boolean;
    Extensions: TypeDefExtensions | undefined;
    Examples: Example[];
    Format: string;
    /** JSON Schema content vocabulary contentMediaType value (e.g. "application/json") */
    ContentMediaType: string;
    Discriminator?: Discriminator;
    IsUnionOpen: boolean;
    Validations: Validations;
    OutputLocation: string;
    ResolvedModel: string;
    IsEmpty: () => boolean;
    IsPrimitive: () => boolean;
    IsCustomType: () => boolean;
    IsCustomClass: () => boolean;
    IsContainer: () => boolean;
    IsPrimitiveContainer: () => boolean;
    IsRequest: () => boolean;
    GetRegistrationID: () => string;
    GetRegistrationIDOrType: () => string;
    IsEqualType: (a: TypeDef) => boolean;
    IsTypeWithFields(): boolean;
    IsStructurallyDeduplicated: () => boolean;
    FindFieldByName(fieldName: string): FieldDef | null;
    EventStreamEnvelope: boolean;
    ResponseEnvelope: boolean;
    IsSerializationWrapperType: boolean;
    JSON: () => Record<string, any>;

    /** Only relevant for event streams, the sentinel value that indicates the end of the stream */
    EventStreamSentinel: string;

    /** Whether this type is used in a union, used for resolving type conflicts */
    UsedInUnion: boolean;
    /** Whether this type is reachable from a request */
    UsedInRequest: boolean;
    /** Whether this type is reachable from a response */
    UsedInResponse: boolean;
    /** Whether this type is reachable from a webhook */
    UsedInWebhook: boolean;
    /** Whether this type is reachable from a callback */
    UsedInCallback: boolean;
    /** Whether this type is reachable from security */
    UsedInSecurity: boolean;
    IsMultipartFile: boolean;
    /** The name of the discriminator property that is pre-applied to the type */
    DiscriminatorPreApplied: string;

    /**
     * Returns a deep copy of the TypeDef.
     */
    Clone: () => TypeDef;

    /**
     * Returns true if the TypeDef has the x-speakeasy-entity extension and the
     * given entityName is found.
     */
    HasEntityName: (entityName: string) => boolean;

    /**
     * Recursively searches the TypeDef tree for a node with the x-speakeasy-entity
     * extension matching the given entityName. Returns undefined if not found.
     */
    FindEntityTypeDef: (entityName: string) => TypeDef | undefined;

    /**
     * Returns true if the TypeDef maps to a primitive Terraform schema attribute
     * (StringAttribute, BoolAttribute, Int64Attribute, NumberAttribute).
     */
    IsTerraformPrimitiveType: () => boolean;

    /**
     * Returns true if any node in the TypeDef tree has a DataType not supported
     * for Terraform import state operations.
     */
    TerraformHasInvalidImportTypes: () => boolean;

    /**
     * Returns true if the receiver (schema) TypeDef has any write-only fields
     * at any nesting depth relative to the given SDK TypeDef. A field is
     * "write-only" when it exists in the schema type but has no name-equivalent
     * in the SDK type.
     */
    TerraformHasNestedWriteOnlyFields: (sdkType: TypeDef) => boolean;

    /**
     * Walks the TypeDef tree and returns all nodes with DataTypes not supported
     * for Terraform import state operations. Returns nil if all types are valid.
     */
    TerraformInvalidImportTypes: () => TerraformInvalidImportType[] | undefined;
  };

  type ParamDef = {
    Field: FieldDef;
    Hidden: boolean;
    AllowEmptyValue: boolean;
    Examples?: Example[];
  };

  type RequestParams = {
    QueryParams?: ParamDef[];
    PathParams?: ParamDef[];
    HeaderParams?: ParamDef[];
    HasQueryParams: () => boolean;
    HasPathParams: () => boolean;
    HasHeaderParams: () => boolean;
  };

  type RequestDef = {
    Field?: FieldDef;
    RequestBody?: FieldDef;
    RequestBodySerializationWrapper?: TypeDef;
    IsRequestBody: boolean;
    Params?: RequestParams;
    Examples?: Example[];
    IsRequestBodyRequired: boolean;
    MatchedContentTypes?: string[];
  };

  type ResponseBodyContent = {
    SerializationMethod: ResponseBodyContentSerializationMethod;
    ContentType: string;
    Content?: FieldDef;
    UsageExample: boolean;
    Examples?: Example[];
    ContentDeserializationWrapper?: TypeDef;
    SSESentinel?: string;
  };

  type ResponseBodyContentSerializationMethod =
    | "json"
    | "raw"
    | "multipart"
    | "form"
    | "string"
    | "eventstream"
    | "jsonl";

  type SubResponse = {
    // either
    // * len(Code) == 1 and len(Content) >= 0
    // OR
    // * len(Code) > 1 and len(Content) == 0 or 1

    Code: string[];
    Headers: boolean;
    Content: ResponseBodyContent[];
    Error: boolean;
  };

  type ResponseDef = {
    Type?: TypeDef;

    // openapi responses are grouped according to the
    // comments in SubResponse so that status codes that return
    // the same schema (or no content) are contained within
    // the same SubResponse

    // TODO there is some other interaction with x-speakeasy error statuses
    // that affects response grouping that would be good to capture
    // here

    // if not all response status codes match x-speakeasy-error status and a
    // response status code is an x-speakeasy-error then an Error object will
    // be generated for use across all paths and the SubResponse.Content for
    // that response status code will be that Error object. In langs that throw exceptions
    // that object will be throwable.

    // if all response status codes match x-speakeasy-error statuses then TODO

    // responses are ordered
    // * so that a response with `default` status code is last
    // * otherwise by the lowest status code in the status codes of a response
    Responses: SubResponse[];
    GetErrorStatusCodes: () => string[];

    /**
     * Returns the FieldDef representing the body of the response for Terraform.
     */
    TerraformBodyFieldDef: (entityName: string) => FieldDef | undefined;
  };

  type ServerVariable = {
    Name: string;
    Default: string;
    Type: TypeDef;
    ServerIndex: number;
    Server: Server;
  };

  type Server = {
    ID: string;
    URL: string;
    IsRelative: boolean;
    Comments?: CommentDef;
    Variables?: ServerVariable[];
  };

  type Servers = {
    Servers: Server[];
    Default: string;
    ServerMap: boolean;
    GetDefaultURL(setDefaultVariables: boolean): string;
    GetVariables(): ServerVariable[];
    HasAbsoluteURL(): boolean;
  };

  type BackoffStrategy = {
    InitialInterval?: number;
    MaxInterval?: number;
    Exponent?: number;
    MaxElapsedTime?: number;
  };

  type Retries = {
    Strategy: string;
    Backoff?: BackoffStrategy;
    StatusCodes: string[];
    RetryConnectionErrors?: boolean;
  };

  /**
   * Resolved SSE-overload stream discriminator. Identifies where the boolean
   * stream toggle lives on the operation.
   */
  type SSEOverloadConfig = {
    /** "body" | "query" */
    In: string;
    /** Resolved field name (currently always "stream"). */
    Name: string;
  };

  type WindowStrategy = {
    Rate: number;
    Period?: string;
  };

  type RateLimit = {
    Strategy: string;
    SlidingWindow?: WindowStrategy;
    Identifier: string;
    Description?: string;
  };

  type OAuth2BuiltInFlow = "client_credentials" | "password";

  type OAuth2Flow = OAuth2BuiltInFlow | "authorization_code" | "implicit";

  type OAuth2Scope = {
    Name: string;
    Comments: CommentDef;
  };

  type OAuth2FlowConfig = {
    Flow: OAuth2Flow;
    Enabled: boolean;
    Comments: CommentDef;
    RequiredScopes: string[];
    AvailableScopes: OAuth2Scope[];
  };

  type OAuth2Config = Record<string, OAuth2FlowConfig>;

  /**
   * Describes Operation extension configurations.
   */
  type OperationExtensions = {
    /**
     * Mapping of all extensions, including those not represented in other
     * properties.
     */
    All: Record<string, any>;

    /**
     * x-speakeasy-exports entries declared on the operation; these alias the
     * operation's generated request type.
     */
    PublicExports?: PublicExport[];

    /**
     * x-speakeasy-docs-rate-limit extension configuration.
     */
    DocsRateLimits: RateLimit[];

    /**
     * x-speakeasy-entity-operation extension configuration.
     */
    EntityOperation?: ExtEntityOperationConfig;

    /**
     * x-speakeasy-entity-missing-codes extension configuration.
     */
    EntityMissingCodes?: number[];

    /**
     * x-speakeasy-mcp extension configuration.
     */
    MCP?: MCP;

    /**
     * x-speakeasy-go-optional-method-arguments extension configuration.
     */
    GoOptionalMethodArguments?: GojaEnum<
      "pointers" | "shared-options" | "method-options"
    >;

    /**
     * x-speakeasy-name-override extension configuration.
     */
    MethodNameOverride: string;

    /**
     * x-speakeasy-pagination extension configuration.
     */
    Pagination?: PaginationConfig;

    /**
     * x-speakeasy-polling extension configuration.
     */
    Polling?: Polling;

    /**
     * x-speakeasy-react-hook extension configuration.
     */
    ReactHook?: ReactHook;

    /**
     * x-speakeasy-retries extension configuration.
     */
    Retries?: Retries;

    /**
     * Resolved SSE-overload stream discriminator. Present when the operation
     * has SSE overload enabled (either via x-speakeasy-sse-overload or inferred).
     */
    SSEOverload?: SSEOverloadConfig;

    /**
     * Usage example if x-speakeasy-usage-example extension configuration is
     * enabled.
     */
    UsageExample?: UsageExampleConfig;
  };

  type Operation = {
    BaseOperation: {
      ID: string;
      OriginalID: string;
      Request?: RequestDef;
      Response: ResponseDef;
      UsesUserAgentHeader: boolean;
    };
    ID: string;
    OriginalID: string;
    Request?: RequestDef;
    Response: ResponseDef;
    UsesUserAgentHeader: boolean;
    Path: string;
    Method: string;
    Security?: FieldDef;
    GlobalSecurity?: FieldDef;
    HoistedSecurityConfig?: HoistedSecurityConfig;
    OAuth2Config?: OAuth2Config;
    Scope: Scope;
    Webhook?: Webhook;
    Servers?: Servers;
    Comments?: CommentDef;
    Tags?: string[];
    Callbacks: TypeDef[];
    OwningSDK: SDK;
    Extensions: OperationExtensions;
    SerializationMethod?: string;
    Globals?: TypeDef;
    MaxMethodParams: number;
    Arguments: {
      Flattening: "none" | "all" | "body" | "params";
      Sorted: FieldDef[];
      ParamFields: FieldDef[];
      BodyFields: FieldDef[];
      BodyField: FieldDef | null;
      IsBodyField(field: FieldDef): boolean;
      IsFieldInBody(field: FieldDef): boolean;
      IsFieldInParams(field: FieldDef): boolean;
    };
    Location?: OpenAPILocation;
    GetID(): string;
    GetAcceptTypes(): string[];
    GetExampleSeed(): number;
    /**
     * Returns the locations of values marked sensitive in the request body,
     * or null when it holds none.
     */
    SensitiveRequestBodyGraph(): SensitiveBodyGraph | null;
    /**
     * Returns the locations of values marked sensitive in all response
     * bodies, including errors, or null when they hold none.
     */
    SensitiveResponseBodyGraph(): SensitiveBodyGraph | null;
    HasSensitiveBodies(): boolean;
  };

  /**
   * Locations of values marked sensitive in an operation's bodies. Nodes[0]
   * is a sentinel for "nothing sensitive below here"; a reference of 0 can be
   * skipped. Roots are the body schemas.
   */
  type SensitiveBodyGraph = {
    Nodes: SensitiveBodyNode[];
    Roots: number[];
  };

  type SensitiveBodyNode = {
    /** Masks the whole value. */
    Sensitive: boolean;
    /**
     * Declared object properties by wire name, sorted by name, including those
     * holding nothing sensitive (Node 0). Unlisted properties use Values.
     */
    Fields?: { Name: string; Node: number }[];
    /** Schema of array items. */
    Item: number;
    /** Schema of map values and additional properties. */
    Values: number;
    /** Union members. */
    Variants?: number[];
    /** Union members of each JSON type that hold nothing sensitive. */
    PlainObject: boolean;
    PlainArray: boolean;
    PlainString: boolean;
    PlainNumber: boolean;
    PlainBoolean: boolean;
  };

  type OpenAPILocation = {
    Line: number;
    Column: number;
  };

  /** A webhook configuration indicates that the operation is a webhook operation */
  type Webhook = {
    /** A webhook key is used to identify the PathItemObject in the OpenAPI document, similar to the `Operation.Path` but for webhooks */
    Key: string;
    Security?: WebhookSecurity;
  };

  type WebhookSecurity = {
    Type: "signature" | "custom";
    HeaderName: string;
    SignatureTextEncoding: "base64" | "base64url" | "hex";
    SignatureAlgorithm: "hmac-sha256";
    ConsumerShouldProvideSecret: Pointer<boolean>;
  };

  type ArazzoInvocationContext = {
    SDK: SDK | null;
    Operation: Operation | null;
    StepIdx: number;
    StepID: string;
  };

  type ArazzoOperationStep = {
    Type: "operation";
    Invocation?: ArazzoInvocationContext;
    UsageContext: UsageContext | null;
    Operation: Operation | null;
    StepIdx: number;
    StepID: string;
    ResponseContentType: string;
    Security?: Example;
  };

  type Test = {
    Name: string;
    Workflow: ArazzoWorkflow | null;
    UsingMockServer: boolean;
    Incomplete?: string[];
    InternalID: string;
    InternalEnvVars?: { Name: string; Value: string }[];
  };

  type ArazzoWorkflow = {
    Name: string;
    Description: string;
    Server: string;
    Security?: Example;
    Steps?: ArazzoStep[];
    Inputs?: FieldDef;
    Outputs?: FieldDef;
  };

  type ArazzoWorkflowStep = {
    Type: "workflow";
    StepID: string;
    StepIdx: number;
    WorkflowID: string;
    Workflow: ArazzoWorkflow | null;
  };

  type ArazzoStep = ArazzoOperationStep | ArazzoWorkflowStep;

  type Tests = {
    TestGroups?: TestGroup[];
    GenerateExampleFile: boolean;
  };

  type TestGroup = {
    Name: string;
    Tests?: Test[];
  };

  type AssertionType = "equal" | "notEqual";

  type AssertionTarget = "statusCode" | "responseBody";

  type ResponseBodyAssertion = {
    Path: string;
    Value: Example;
    Content: ResponseBodyContent;
  };

  type Assertion = {
    Type: AssertionType;
    TargetType: AssertionTarget;
    Target?: any;
    Value: string | ResponseBodyAssertion;
  };

  /**
   * Collection of Assertion.
   */
  type Assertions = Assertion[];

  type CLICommandBind = {
    In: string;
    Pointer: string;
    Mode: string;
  };

  type CLICommandInput = {
    ID: string;
    Name: string;
    Summary: string;
    Type: string;
    Required: boolean;
    Variadic: boolean;
    Shorthand: string;
    Default: any;
    DefaultFrom: string;
    Bind: CLICommandBind | null;
    Enum?: any[];
  };

  type CLICommandPreset = {
    Bind: CLICommandBind;
    Value: any;
  };

  type CLIVariantSelectors = {
    Own?: string[];
    Foreign?: string[];
    DiscriminatorKey?: string;
    DiscriminatorValue?: any;
    DiscriminatorAliases?: any[];
  };

  type CLICommandRoute = {
    OperationID: string;
    RequestVariant: string;
    Selectors?: CLIVariantSelectors | null;
  };

  type CLICommandSourceType = "operation" | "group" | "planned" | "custom";

  type CLICommandSource = {
    Type: GojaEnum<CLICommandSourceType>;
    Routes: CLICommandRoute[] | null;
    Note: string;
  };

  type CLICommandHelp = {
    Defaults: string;
    Learn: string;
    Escalate: string;
  };

  type CLICommandProjection = {
    JQ: string;
    Format: string;
  };

  type CLICommandArtifactSegment = {
    Field?: string;
    Wild?: boolean;
  };

  type CLICommandArtifact = {
    ContentPointer: string;
    Segments: CLICommandArtifactSegment[] | null;
    Kind: string;
    DefaultPath: string;
    TypeField: string;
    DataField: string;
    MimeTypeField: string;
    URIField: string;
    IDField: string;
    StatusField: string;
    TerminalStatus: string;
    ResponseCode: string;
  };

  type CLICommandStreamMetadata = {
    Select: string[];
    Pointers: string[];
    Label: string;
    Complete?: string[] | null;
    When?: string;
  };

  type CLICommandStreamProjection = {
    Metadata?: CLICommandStreamMetadata[] | null;
    Select: string;
    Pointer: string;
  };

  type CLICommandOutcome = {
    StateFrom?: string;
    StatePointer?: string;
    StreamStateFrom?: string;
    StreamStatePointer?: string;
    States: Record<string, string>;
    ErrorFrom?: string;
    ErrorPointer?: string;
    StreamErrorFrom?: string;
    StreamErrorPointer?: string;
    ErrorMessageField?: string;
  };

  type CLICommandOutput = {
    Projection: CLICommandProjection | null;
    Artifact?: CLICommandArtifact | null;
    Stream?: CLICommandStreamProjection | null;
    Outcome?: CLICommandOutcome | null;
  };

  type CLICommandAsyncParameter = {
    In: string;
    Name: string;
  };

  type CLICommandAsyncPathSegment = {
    Field: string;
    Index: number;
    IsIndex: boolean;
  };

  type CLICommandAsyncResolvedParameter = {
    In: string;
    Name: string;
    Value: unknown;
  };

  type CLICommandAsyncID = {
    From: string;
    Pointer: string;
    Segments: CLICommandAsyncPathSegment[] | null;
    To: CLICommandAsyncParameter;
  };

  type CLICommandAsync = {
    OperationID: string;
    ID: CLICommandAsyncID;
    Params?: Record<string, unknown> | null;
    ResolvedParams?: CLICommandAsyncResolvedParameter[] | null;
    StateFrom: string;
    StatePointer: string;
    StateSegments: CLICommandAsyncPathSegment[] | null;
    States: Record<string, string>;
    Interval: string;
    Backoff: number;
    MaxInterval: string;
    Timeout: string;
    ErrorField: string;
    ErrorMessageField: string;
    CreateResponseCode: string;
    ResponseCode: string;
  };

  type CLICommandExample = {
    Summary: string;
    Command: string;
  };

  type CLICommand = {
    ID: string;
    Path: string[];
    Category: string;
    Summary: string;
    Tagline: string;
    Description: string;
    Source: CLICommandSource;
    Args: CLICommandInput[] | null;
    Flags: CLICommandInput[] | null;
    Presets: CLICommandPreset[] | null;
    Async?: CLICommandAsync | null;
    Output: CLICommandOutput | null;
    Examples: CLICommandExample[] | null;
    Help?: CLICommandHelp | null;
    Hints?: Record<string, string[]>;
    CatalogDefaults?: {
      CatalogCommand: string;
      Value: string;
      Label: string;
    }[];
  };

  type CLICommandManifest = {
    Version: number;
    Categories?: string[];
    Commands: CLICommand[];
  };

  type CLIErrorRule = {
    Reason: string;
    Type?: string;
    Hints?: string[];
    HasHints: boolean;
  };

  type CLIErrorTypeHints = {
    Type: string;
    Hints: string[];
  };

  type CLIErrorPointerSegment = {
    Field?: string;
    IsWild?: boolean;
  };

  type CLIErrorProbe = {
    Pointer: string;
    Segments?: CLIErrorPointerSegment[];
    Reasons?: CLIErrorRule[];
  };

  type CLIErrorManifest = {
    Version: number;
    /**
     * Declared primary reason carrier: restricted JSONPath, relative to the
     * response body's error object, whose string values are the structured
     * reason codes the top-level rules match. Empty means the default
     * $.reason member; declaring it opts into verbatim reason promotion.
     */
    ReasonPointer?: string;
    ReasonSegments?: CLIErrorPointerSegment[];
    Reasons?: CLIErrorRule[];
    /**
     * Additional reason carriers consulted after the primary carrier, in
     * declaration order; each declares its pointer, so each promotes.
     */
    Probes?: CLIErrorProbe[];
    Types?: CLIErrorTypeHints[];
    /**
     * Normalize a single-element array-wrapped error body
     * ([{"error": {...}}]) to its sole element before classification.
     */
    UnwrapErrorArray?: boolean;
  };

  type AST = {
    ASTVersion: string;
    MainSDK: SDK | null;
    Webhooks: SequencedMap<string, Operation[]>;
    /**
     * Namespaced Arazzo-derived test workflow graph data.
     */
    Arazzo?: { Workflows: ArazzoWorkflow[] };
    Tests?: Tests;
    Components: Record<string, TypeDef | null>;
    UsedFeatures: Record<string, boolean>;
    UsedTypes: Record<string, boolean>;
    BucketedTypes: BucketedTypes;
    OperationServers: SequencedMap<string, Servers | null>;
    PublicExports?: ResolvedPublicExports;

    /**
     * Decoded x-speakeasy-cli-commands manifest: declaratively defined CLI
     * intent commands rendered by the cli target. Only set for that target.
     */
    CLICommands?: CLICommandManifest | null;

    /**
     * Decoded x-speakeasy-cli-errors manifest: global error reason rules and
     * type-hint overrides rendered by the cli target.
     */
    CLIErrors?: CLIErrorManifest | null;

    /**
     * operationId → self-contained JSON Schema for the operation's
     * application/json request body (components bundled under $defs).
     * Rendered by the cli target as an exact --schema surface.
     */
    CLIBodySchemas?: Record<string, string> | null;

    /**
     * Terraform Provider as gathered from various x-speakeasy-entity*
     * extensions across the source document. Only set when generation target
     * is "terraform".
     */
    TerraformProvider?: TerraformProvider;
  };

  type OptionalityReason =
    | "not-optional"
    | "optional-scheme"
    | "operation-override"
    | "env-var";

  type SecurityConfig = {
    OptionalityReason: OptionalityReason;
    Disabled: boolean;
    OAuth2Config: OAuth2Config;
  };

  type SDK = {
    FormattedName: string; // Added when generating SDK docs not part of original AST
    FieldAccessName: string; // Added when generating SDK docs not part of original AST
    FieldName: string;
    Type: TypeDef;
    Group: string;
    Servers?: Servers;
    Comments?: CommentDef;
    SubSDKs: SDK[];
    Security?: FieldDef;
    SecurityConfig: SecurityConfig;
    Operations: Operation[];
    AdditionalTypes: TypeDef[];
    TestGroup: string;
    Globals?: TypeDef;
    FindOperation(operationID: string): Operation | null;
    HasAnyOperationServers(): boolean;
    OutputTests: boolean;
  };

  type UsageContext = {
    Operation: Operation;
    StepIdx: number;
    StepID: string;
    RenderFeature(feature: string, global: boolean): boolean;
    SDK: SDK;
    SkipSDKInstantiation: boolean;
    ContextIndex: number;
    Config?: UsageExampleConfig;
    Scopes?: UsageExampleScope[];
    HandleError?: boolean;
    SkipResponseBodyAssertions?: boolean;
    ExampleName?: string;
    Test?: ArazzoWorkflow;
    Assertions?: Assertions;
    UsingMockServer?: boolean;
    IsMainExample: boolean;
    AsyncMode: boolean;
    PopulateGlobalParameterScopes(
      exampleName: string,
      shouldIncludeServerSelection: boolean,
    );
  };

  type UsageExampleConfig = {
    Title?: string;
    Description?: string;
    Position?: number;
    Tags?: string[];
    Extensions?: string[];
  };

  type UsageExampleScope = {
    OpFilter: string;
    Feature: string;
    IsGlobal: boolean;
    Value?: any;
  };

  type PaginationType = "offsetLimit" | "cursor" | "url";
  type PaginationInputInType = "parameters" | "requestBody";
  type PaginationInputType = "limit" | "offset" | "page" | "cursor";
  type PaginationInputs = {
    Name: string;
    In: GojaEnum<PaginationInputInType>;
    Type: GojaEnum<PaginationInputType>;
    Optional: boolean;
  };
  type PaginationOutputs = {
    CanUseDotNotation: boolean;

    Results: string;
    ResultsDot: string;

    NumPages: string;
    NumPagesDot: string;

    NextCursor: string;
    NextCursorDot: string;

    NextURL: string;
    NextURLDot: string;

    StatusCode: string;
    ContentType: string;
  };
  type PaginationConfig = {
    Type: GojaEnum<PaginationType>;
    Inputs: PaginationInputs[];
    Outputs: PaginationOutputs;
  };

  /**
   * Describes x-speakeasy-polling extension configuration.
   */
  type Polling = {
    /**
     * Collection of polling options.
     */
    Options: PollingOption[];
  };

  /**
   * Describes an individual polling option from the x-speakeasy-polling
   * extension configuration.
   */
  type PollingOption = {
    /**
     * Number of seconds to delay before first polling attempt.
     */
    DelaySeconds: number | undefined;

    /**
     * Criteria for the polling option that should stop polling attempts and
     * immediately return failure.
     */
    FailureCriteria: Assertions;

    /**
     * Number of seconds between polling attempts.
     */
    IntervalSeconds: number | undefined;

    /**
     * Maximum number of polling attempts.
     */
    LimitCount: number | undefined;

    /**
     * Name of the polling option.
     */
    Name: string;

    /**
     * Criteria for the polling option that should stop polling attempts and
     * immediately return success.
     */
    SuccessCriteria: Assertions;
  };

  type ReactHook = {
    Type: "infer" | "query" | "mutation";
    Name: string;
    Disabled: boolean;
    QueryKey?: {
      IncludeRequestBody: boolean;
    };
  };

  type MCP = {
    Disabled: boolean;
    Name: string;
    Description: string;
    Scopes: string[];
    Title: string;
    DestructiveHint: boolean;
    IdempotentHint: boolean;
    OpenWorldHint: boolean;
    ReadOnlyHint: boolean;
  };

  type SDKGenConfigField = {
    Name: string;
    Required: boolean;
    RequiredForPublishing?: boolean;
    DefaultValue?: any;
    Description?: string;
    Language?: string;
    SecretName?: string;
    ValidationRegex?: string;
    ValidationMessage?: string;
  };

  type SDKGenConfigFields = Record<string, SDKGenConfigField>;

  type CallbackIterator<T, U> = (cb: (a: T, b: U) => boolean) => void;

  type SequencedMap<K, V> = {
    Get: (key: K) => [V, boolean];
    Set: (key: K, value: V) => [V, boolean];
    Delete: (key: K) => [V, boolean];
    Len: () => number;
    All: () => CallbackIterator<K, V>;
  };

  type ParameterExamples = {
    Path?: SequencedMap<string, YamlNode>;
    Query?: SequencedMap<string, YamlNode>;
    Header?: SequencedMap<string, YamlNode>;
  };

  type YamlNode = {};

  type OperationExamples = {
    Parameters?: ParameterExamples;
    RequestBody?: SequencedMap<string, YamlNode>;
    Responses?: SequencedMap<string, SequencedMap<string, YamlNode> | null>;
  };

  type Examples = SequencedMap<
    string,
    SequencedMap<string, OperationExamples> | null
  >;

  /** Represents an individual command to be ran by the generator. */
  type RunnerCommand = {
    /** Command name. */
    command: string;

    /** Arguments to pass to the command.
     *
     * If runtime argument variables are given, each argument is ran through
     * [text/template] to enable variable substition. For example, templating
     * an Example variable via {{ .Example }} anywhere in the string.
     *
     * Available variables are:
     *
     *  - PackageVersion: Generator selected new version for target.
     *
     * It is highly recommended add any templating variables in the runner
     * dependencies to prevent confusing errors. */
    args?: string[];

    /** Mapping of environment variable names to values for overriding. */
    environmentVariables?: Record<string, string>;

    /** When enabled, return without error if the command is not found. This
     * allows optional tooling commands to be ran, but only if present. */
    ignoreCommandNotFound?: boolean;

    /** When enabled, discards stdout output. */
    stdoutDiscard?: boolean;

    /** When enabled, consecutive commands with parallel set to true will be
     * ran concurrently. All parallel commands in a group must complete before
     * the next non-parallel command runs. */
    parallel?: boolean;
  };

  /** Represents a list of commands to be ran by the generator. */
  type RunnerCommands = RunnerCommand[];

  /** An individual command dependency. */
  type RunnerCommandDependency = {
    /** Command name. */
    command: string;

    /** Documentation about how to install the command. */
    installDocumentation: string;

    /** Verify the dependency version. */
    version?: RunnerCommandVersionDependency;
  };

  /** Represents a list of commands to be ran by the generator. */
  type RunnerCommandDependencies = RunnerCommandDependency[];

  /** Command version dependency verification. */
  type RunnerCommandVersionDependency = {
    /** Arguments to pass to the command to retrieve the version. */
    args: string[];

    /** Minimum supported version for the command. */
    minVersion: string;

    /** Regular expression to extract the version for the command. */
    regex: string;
  };

  /** Dependencies configuration for the runner. */
  type RunnerDependencies = {
    /** Mapping of required argument variables names to RE2 regular expressions
     * for argument templating. */
    argVariables?: Record<string, string>;

    /** Required command dependencies for the runner. */
    commands: RunnerCommandDependencies;

    /** Mapping of required environment variables names to RE2 regular
     * expressions value checks. */
    environmentVariables?: Record<string, string>;
  };

  /** Configuration for a generator processrunner.Runner. */
  type RunnerConfiguration = {
    /** Commands to be ran. */
    commands: RunnerCommands;

    /** Dependencies for the runner. */
    dependencies: RunnerDependencies;
  };

  /** Represents target-specific configuration for compiling generated code. */
  type CompileConfiguration = {
    /** Runner configuration for compilation. */
    runner: RunnerConfiguration;
  };

  /** Represents target-specific configuration for linting generated code. */
  type LintConfiguration = {
    /** Runner configuration for linting. */
    runner: RunnerConfiguration;
  };

  /** Represents target-specific configuration for testing generated code. */
  type TestingConfiguration = {
    /** Processes and dependencies necessary to compile the generated target
     * testing code. */
    compile?: CompileConfiguration;

    /** Processes and dependencies necessary to run the generated target testing
     * code. */
    runner?: RunnerConfiguration;
  };

  /** Represents the initial target implementation configuration presented to
   * the generator from the target. */
  type TargetInitialConfiguration = {
    /** Generated code compilation configuration. */
    compile?: CompileConfiguration;

    /** Generated code linting configuration. */
    lint?: LintConfiguration;

    /** Generated code testing configuration. */
    testing?: TestingConfiguration;
  };

  /** Go representation of runtime platforms, e.g. from the GOOS environment
   * variable. */
  type GoRuntimePlatform = "darwin" | "linux" | "windows";

  /** Represents the initial configuration presented to the target from the
   * generator. */
  type GeneratorInitialConfiguration = {
    /** Runtime environment variables. */
    Env: Record<string, string>;

    /** Target-specific configuration passed by customer. */
    LangCfg: Record<string, any>;

    /** Runtime platform. */
    Platform: GoRuntimePlatform;

    /** Enabled when the generation includes publishing. */
    Publish: boolean;

    /** URL of the repository for the generated code. */
    RepoURL: string;
  };

  /** Represents a parameter usage example, e.g. global parameter example for
   *  testing SDK client instantiation. */
  type ParameterUsage = {
    /** Parameter example. */
    example: any;

    /** Parameter field. */
    field: FieldDef;
  };

  type HTTPClientUsage = {
    type: "usage" | "test";
    testName?: string;
  };

  type ExampleReferenceValue = {
    isExampleReferenceValue: true;
    path: string;
    parents: FieldDef[];
    source: FieldDef;
    replacements: ExampleReplacement[];
  };
}
