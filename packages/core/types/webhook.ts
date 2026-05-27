export interface Webhook {
  id: string;
  workspace_id: string;
  url: string;
  has_secret: boolean;
  events: string[];
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateWebhookRequest {
  url: string;
  secret?: string;
  events: string[];
  active?: boolean;
}

export interface UpdateWebhookRequest {
  url?: string;
  secret?: string;
  events?: string[];
  active?: boolean;
}
