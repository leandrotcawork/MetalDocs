export type ApprovalState =
  | 'draft'
  | 'under_review'
  | 'approved'
  | 'scheduled'
  | 'published'
  | 'superseded'
  | 'rejected'
  | 'obsolete'
  | 'cancelled';

export type SignatureMethod = 'password_reauth' | 'icp_brasil';
export type QuorumKind = 'any_1' | 'all_of' | 'm_of_n';
export type DriftPolicy = 'auto_cancel' | 'alert_only' | 'none';

export interface RouteStage {
  label: string;
  members: string[];
  quorum_kind: QuorumKind;
  m?: number;
  drift_policy: DriftPolicy;
}

export interface Route {
  id: string;
  name: string;
  tenant_id: string;
  profile_code: string;
  stages: RouteStage[];
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Signoff {
  id: string;
  actor_user_id: string;
  decision: 'approve' | 'reject';
  reason?: string;
  signature_method: SignatureMethod;
  signed_at: string;
}

export interface StageInstance {
  id: string;
  stage_index: number;
  label: string;
  status: 'pending' | 'active' | 'passed' | 'failed' | 'cancelled';
  signoffs: Signoff[];
}

export interface ApprovalInstance {
  id: string;
  document_id: string;
  route_id: string;
  status: 'in_progress' | 'completed' | 'cancelled';
  submitted_by: string;
  submitted_at: string;
  completed_at?: string;
  stages: StageInstance[];
  etag?: string;
}

export interface InboxItem {
  instance_id: string;
  document_id: string;
  document_title: string;
  area_code: string;
  submitted_by: string;
  submitted_at: string;
  stage_label: string;
  quorum_progress: string;
}

export interface SubmitRequest {
  route_id: string;
  content_hash: string;
}

export interface SubmitResponse {
  instance_id: string;
  was_replay: boolean;
  etag: string;
}

export interface SignoffRequest {
  decision: 'approve' | 'reject';
  reason?: string;
  password: string;
  content_hash: string;
}

export interface SignoffResponse {
  signoff_id: string;
  was_replay: boolean;
}

export interface PublishRequest {
  content_hash: string;
}

export interface PublishResponse {
  document_id: string;
}

export interface SchedulePublishRequest {
  content_hash: string;
  effective_from: string;
}

export interface SchedulePublishResponse {
  document_id: string;
  scheduled_at: string;
}

export interface SupersedeRequest {
  content_hash: string;
  supersedes_document_id: string;
}

export interface SupersedeResponse {
  document_id: string;
}

export interface ObsoleteRequest {
  reason: string;
}

export interface ObsoleteResponse {
  document_id: string;
}

export interface CancelRequest {
  reason: string;
}

export interface CancelResponse {
  document_id: string;
}

export interface CreateRouteRequest {
  name: string;
  profile_code: string;
  stages: RouteStage[];
}

export interface CreateRouteResponse {
  route_id: string;
}

export interface UpdateRouteRequest {
  name?: string;
  stages?: RouteStage[];
}

export interface UpdateRouteResponse {
  route_id: string;
}

export interface DeactivateRouteResponse {
  route_id: string;
}

export interface ListInboxParams {
  area_code?: string;
  limit?: number;
  offset?: number;
}

export interface ListInboxResponse {
  items: InboxItem[];
  total: number;
}

export interface ListRoutesResponse {
  routes: Route[];
  total: number;
}
