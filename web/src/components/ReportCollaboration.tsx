import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useState } from "react";
import { api } from "../api/client";
import type {
  CreatedPublicReportLink,
  PublicReportLink,
  WorkspaceRole,
} from "../api/types";

const writableRoles = new Set<WorkspaceRole>(["owner", "admin", "operator"]);

export function ReportCollaboration({
  runID,
  role,
  userID,
}: {
  runID: string;
  role: WorkspaceRole;
  userID: string;
}) {
  const queryClient = useQueryClient();
  const [body, setBody] = useState("");
  const [revealed, setRevealed] = useState<CreatedPublicReportLink | null>(
    null,
  );
  const canWrite = writableRoles.has(role);
  const comments = useQuery({
    queryKey: ["run-comments", runID],
    queryFn: () => api.runComments(runID),
  });
  const links = useQuery({
    queryKey: ["run-public-links", runID],
    queryFn: () => api.publicReportLinks(runID),
  });
  const addComment = useMutation({
    mutationFn: () => api.createRunComment(runID, body.trim()),
    onSuccess: async () => {
      setBody("");
      await queryClient.invalidateQueries({
        queryKey: ["run-comments", runID],
      });
    },
  });
  const deleteComment = useMutation({
    mutationFn: (commentID: string) => api.deleteRunComment(runID, commentID),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["run-comments", runID] }),
  });
  const createLink = useMutation({
    mutationFn: () => api.createPublicReportLink(runID),
    onSuccess: async (created) => {
      setRevealed(created);
      await queryClient.invalidateQueries({
        queryKey: ["run-public-links", runID],
      });
    },
  });
  const revokeLink = useMutation({
    mutationFn: (linkID: string) => api.revokePublicReportLink(runID, linkID),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: ["run-public-links", runID],
      }),
  });

  function submitComment(event: FormEvent) {
    event.preventDefault();
    if (body.trim()) addComment.mutate();
  }

  const shareURL = revealed
    ? `${window.location.origin}/shared/${revealed.token}`
    : "";

  return (
    <section className="report-collaboration">
      <div className="collaboration-heading">
        <div>
          <p className="section-label">08 / Collaboration record</p>
          <h2>Notes &amp; public evidence</h2>
        </div>
        <span>{role} access</span>
      </div>

      <div className="collaboration-grid">
        <div className="report-comments">
          <div className="collaboration-subhead">
            <h3>Team comments</h3>
            <span>{comments.data?.length ?? 0}</span>
          </div>
          {canWrite && (
            <form onSubmit={submitComment}>
              <textarea
                value={body}
                onChange={(event) => setBody(event.target.value)}
                placeholder="Add an operational note to this report…"
                maxLength={2000}
                rows={3}
                required
              />
              <button
                type="submit"
                disabled={addComment.isPending || !body.trim()}
              >
                {addComment.isPending ? "Recording…" : "Record comment"}
              </button>
            </form>
          )}
          {comments.error && (
            <p className="field-error">{comments.error.message}</p>
          )}
          <div className="comment-ledger">
            {comments.data?.map((comment) => {
              const canDelete =
                comment.authorId === userID ||
                role === "owner" ||
                role === "admin";
              return (
                <article key={comment.id}>
                  <header>
                    <div>
                      <strong>{comment.authorName}</strong>
                      <span>{comment.authorEmail}</span>
                    </div>
                    <time dateTime={comment.createdAt}>
                      {new Date(comment.createdAt).toLocaleString()}
                    </time>
                  </header>
                  <p>{comment.body}</p>
                  {canDelete && (
                    <button
                      type="button"
                      onClick={() => deleteComment.mutate(comment.id)}
                      disabled={deleteComment.isPending}
                    >
                      Delete
                    </button>
                  )}
                </article>
              );
            })}
            {!comments.isLoading && comments.data?.length === 0 && (
              <p className="empty-collaboration">
                No team notes have been recorded.
              </p>
            )}
          </div>
        </div>

        <div className="public-links">
          <div className="collaboration-subhead">
            <h3>Public links</h3>
            {canWrite && (
              <button
                type="button"
                onClick={() => createLink.mutate()}
                disabled={createLink.isPending}
              >
                {createLink.isPending ? "Publishing…" : "Publish report"}
              </button>
            )}
          </div>
          {revealed && (
            <div className="share-reveal" role="status">
              <strong>Public report ready</strong>
              <input aria-label="Public report URL" readOnly value={shareURL} />
              <div>
                <button
                  type="button"
                  onClick={() => void navigator.clipboard?.writeText(shareURL)}
                >
                  Copy link
                </button>
                <a href={shareURL} target="_blank" rel="noreferrer">
                  Open
                </a>
                <button type="button" onClick={() => setRevealed(null)}>
                  Done
                </button>
              </div>
            </div>
          )}
          {links.error && <p className="field-error">{links.error.message}</p>}
          <div className="share-ledger">
            {links.data?.map((link) => (
              <PublicLinkRow
                key={link.id}
                link={link}
                canWrite={canWrite}
                busy={revokeLink.isPending}
                onRevoke={() => revokeLink.mutate(link.id)}
              />
            ))}
            {!links.isLoading && links.data?.length === 0 && (
              <p className="empty-collaboration">
                This report has no public links.
              </p>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}

function PublicLinkRow({
  link,
  canWrite,
  busy,
  onRevoke,
}: {
  link: PublicReportLink;
  canWrite: boolean;
  busy: boolean;
  onRevoke: () => void;
}) {
  const [referenceTime] = useState(Date.now);
  const status = link.revokedAt
    ? "revoked"
    : new Date(link.expiresAt).getTime() <= referenceTime
      ? "expired"
      : "active";
  return (
    <article>
      <div>
        <code>{link.tokenPrefix}…</code>
        <span className={`key-status is-${status}`}>{status}</span>
      </div>
      <dl>
        <div>
          <dt>Expires</dt>
          <dd>{new Date(link.expiresAt).toLocaleDateString()}</dd>
        </div>
        <div>
          <dt>Last viewed</dt>
          <dd>
            {link.lastViewedAt
              ? new Date(link.lastViewedAt).toLocaleString()
              : "Never"}
          </dd>
        </div>
      </dl>
      {canWrite && status === "active" && (
        <button type="button" onClick={onRevoke} disabled={busy}>
          Revoke
        </button>
      )}
    </article>
  );
}
