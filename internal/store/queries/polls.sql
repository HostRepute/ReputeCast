-- Polls and surveys (internal/polls, design expansion M21).

-- name: PollsUpsert :one
-- A campaign's poll as its content says now (a test send or a send needs
-- its links). The number in signed links never changes.
INSERT INTO polls (id, workspace_id, campaign_id, block_id, kind, question, answers, settings, closes_at, updated_at)
VALUES (@id, @workspace_id, @campaign_id, @block_id, @kind, @question, @answers, @settings, sqlc.narg(closes_at), @now)
ON CONFLICT (campaign_id, block_id) WHERE campaign_id IS NOT NULL DO UPDATE SET
    kind = EXCLUDED.kind, question = EXCLUDED.question, answers = EXCLUDED.answers, settings = EXCLUDED.settings,
    -- A poll a person reopened keeps the closing they chose.
    closes_at = CASE WHEN polls.reopened_at IS NOT NULL THEN polls.closes_at ELSE EXCLUDED.closes_at END,
    updated_at = EXCLUDED.updated_at
RETURNING id, number;

-- name: PollsOfCampaign :many
SELECT p.id, p.workspace_id, p.number, p.campaign_id, p.block_id, p.kind, p.question, p.answers, p.settings, p.closes_at, p.closed_at, p.closed_by,
       p.reopened_at, p.created_at, p.updated_at, coalesce(u.name, '')::text AS closed_by_name
FROM polls p
LEFT JOIN users u ON u.id = p.closed_by
WHERE p.workspace_id = @workspace_id AND p.campaign_id = @campaign_id
ORDER BY p.created_at;

-- name: PollByNumber :one
-- The poll a signed link names, with what its pages show.
SELECT p.id, p.workspace_id, p.number, p.campaign_id, p.block_id, p.kind, p.question, p.answers, p.settings, p.closes_at, p.closed_at,
       coalesce(c.name, '')::text AS campaign_name, c.started_at AS campaign_started_at, coalesce(c.status, '')::text AS campaign_status
FROM polls p
LEFT JOIN campaigns c ON c.id = p.campaign_id
WHERE p.number = @number;

-- name: PollByID :one
SELECT p.id, p.workspace_id, p.number, p.campaign_id, p.block_id, p.kind, p.question, p.answers, p.settings, p.closes_at, p.closed_at, p.closed_by,
       p.reopened_at, p.created_at, p.updated_at, coalesce(u.name, '')::text AS closed_by_name,
       coalesce(c.name, '')::text AS campaign_name, c.started_at AS campaign_started_at, coalesce(c.status, '')::text AS campaign_status
FROM polls p
LEFT JOIN users u ON u.id = p.closed_by
LEFT JOIN campaigns c ON c.id = p.campaign_id
WHERE p.workspace_id = @workspace_id AND p.id = @id;

-- name: PollSetClosed :exec
UPDATE polls SET closed_at = sqlc.narg(closed_at), closed_by = sqlc.narg(closed_by), reopened_at = sqlc.narg(reopened_at), updated_at = @now
WHERE id = @id;

-- name: PollSetClosesAt :exec
UPDATE polls SET closes_at = sqlc.narg(closes_at), updated_at = @now WHERE id = @id;

-- name: PollsOfWorkspace :many
-- Every poll of the workspace (the automation trigger's picker, MCP).
SELECT p.id, p.number, p.campaign_id, p.block_id, p.kind, p.question, p.answers, p.closes_at, p.closed_at,
       coalesce(c.name, '')::text AS campaign_name, coalesce(c.status, '')::text AS campaign_status, c.started_at AS campaign_started_at
FROM polls p
LEFT JOIN campaigns c ON c.id = p.campaign_id
WHERE p.workspace_id = @workspace_id
ORDER BY coalesce(c.started_at, p.created_at) DESC;

-- name: PollAnswerOfContact :one
SELECT id, poll_id, workspace_id, contact_id, message_id, kind, test_email, answer, first_answer, changes, comment, commented_at, client, answered_at, updated_at
FROM poll_answers
WHERE poll_id = @poll_id AND contact_id = @contact_id AND kind = 'contact'
FOR UPDATE;

-- name: PollAnswerOfTester :one
SELECT id, poll_id, workspace_id, contact_id, message_id, kind, test_email, answer, first_answer, changes, comment, commented_at, client, answered_at, updated_at
FROM poll_answers
WHERE poll_id = @poll_id AND test_email = @test_email AND kind = 'test'
FOR UPDATE;

-- name: PollAnswerByID :one
SELECT id, poll_id, workspace_id, contact_id, message_id, kind, test_email, answer, first_answer, changes, comment, commented_at, client, answered_at, updated_at
FROM poll_answers
WHERE id = @id
FOR UPDATE;

-- name: PollAnswerInsert :exec
INSERT INTO poll_answers (id, poll_id, workspace_id, contact_id, message_id, kind, test_email, answer, first_answer, client, answered_at, updated_at)
VALUES (@id, @poll_id, @workspace_id, sqlc.narg(contact_id), sqlc.narg(message_id), @kind, @test_email, @answer, @answer, @client, @at, @at);

-- name: PollAnswerChange :exec
UPDATE poll_answers SET answer = @answer, changes = changes + 1, message_id = coalesce(sqlc.narg(message_id), message_id), client = @client, updated_at = @at
WHERE id = @id;

-- name: PollAnswerComment :exec
UPDATE poll_answers SET comment = @comment, commented_at = @at, updated_at = @at WHERE id = @id;

-- name: PollAnswerEvent :exec
INSERT INTO poll_answer_events (answer_id, poll_id, kind, answer, previous, at)
VALUES (@answer_id, @poll_id, @kind, sqlc.narg(answer), sqlc.narg(previous), @at);

-- name: PollAutomatedBump :exec
INSERT INTO poll_automated_visits (poll_id, reason, n, last_at) VALUES (@poll_id, @reason, 1, @at)
ON CONFLICT (poll_id, reason) DO UPDATE SET n = poll_automated_visits.n + 1, last_at = EXCLUDED.last_at;

-- name: PollAutomated :many
SELECT reason, n FROM poll_automated_visits WHERE poll_id = @poll_id;

-- name: PollTestRecipientInsert :exec
INSERT INTO poll_test_recipients (id, workspace_id, campaign_id, email, created_at) VALUES (@id, @workspace_id, @campaign_id, @email, @at);

-- name: PollTestRecipient :one
SELECT id, workspace_id, campaign_id, email FROM poll_test_recipients WHERE id = @id;

-- name: PollTestAnswers :many
SELECT a.id, a.poll_id, a.test_email, a.answer, a.updated_at, p.block_id
FROM poll_answers a
JOIN polls p ON p.id = a.poll_id
WHERE p.workspace_id = @workspace_id AND p.campaign_id = @campaign_id AND a.kind = 'test'
ORDER BY a.updated_at;

-- name: PollTestAnswersClear :execrows
DELETE FROM poll_answers a USING polls p
WHERE p.id = a.poll_id AND p.workspace_id = @workspace_id AND p.campaign_id = @campaign_id AND a.kind = 'test';

-- name: PollSummary :one
-- A poll's headline numbers: contacts' answers, non-contacts' and tests
-- apart, how many changed their answer and wrote a comment.
SELECT count(*) FILTER (WHERE kind = 'contact')::bigint AS contacts,
       count(*) FILTER (WHERE kind = 'non_contact')::bigint AS non_contacts,
       count(*) FILTER (WHERE kind = 'test')::bigint AS tests,
       count(*) FILTER (WHERE kind = 'contact' AND changes > 0)::bigint AS changed,
       count(*) FILTER (WHERE kind <> 'test' AND comment <> '')::bigint AS comments,
       min(answered_at) FILTER (WHERE kind = 'contact') AS first_at,
       max(updated_at) FILTER (WHERE kind = 'contact') AS last_at
FROM poll_answers
WHERE poll_id = @poll_id;

-- name: PollAnswerCounts :many
SELECT answer, count(*)::bigint AS n
FROM poll_answers
WHERE poll_id = @poll_id AND kind = 'contact'
GROUP BY answer
ORDER BY answer;

-- name: PollBreakdownTier :many
SELECT coalesce(c.tier, '')::text AS tier, a.answer, count(*)::bigint AS n
FROM poll_answers a
JOIN contacts c ON c.id = a.contact_id
WHERE a.poll_id = @poll_id AND a.kind = 'contact'
GROUP BY c.tier, a.answer;

-- name: PollBreakdownLanguage :many
SELECT lower(split_part(replace(coalesce(c.language, ''), '_', '-'), '-', 1))::text AS language, a.answer, count(*)::bigint AS n
FROM poll_answers a
JOIN contacts c ON c.id = a.contact_id
WHERE a.poll_id = @poll_id AND a.kind = 'contact'
GROUP BY 1, a.answer;

-- name: PollBreakdownList :many
SELECT l.id AS list_id, l.name AS list_name, a.answer, count(*)::bigint AS n
FROM poll_answers a
JOIN list_memberships m ON m.contact_id = a.contact_id AND m.status = 'subscribed'
JOIN lists l ON l.id = m.list_id AND l.deleted_at IS NULL
WHERE a.poll_id = @poll_id AND a.kind = 'contact'
GROUP BY l.id, l.name, a.answer
ORDER BY l.name;

-- name: PollAnswersByHour :many
-- Answers by the hour after the send started they first came in.
SELECT floor(extract(epoch FROM (a.answered_at - @since::timestamptz)) / 3600)::int AS hour, count(*)::bigint AS n
FROM poll_answers a
WHERE a.poll_id = @poll_id AND a.kind = 'contact' AND a.answered_at >= @since::timestamptz
GROUP BY 1
ORDER BY 1;

-- name: PollComments :many
-- Comments, newest first, with the contact's name (the caller leaves it
-- out for people without subscribers:view, whose search doesn't match
-- names either).
SELECT a.id, a.kind, a.answer, a.comment, coalesce(a.commented_at, a.updated_at) AS at, a.contact_id,
       coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name, coalesce(c.email, '')::text AS email,
       count(*) OVER ()::bigint AS total
FROM poll_answers a
LEFT JOIN contacts c ON c.id = a.contact_id
WHERE a.poll_id = @poll_id AND a.kind <> 'test' AND a.comment <> ''
  AND (a.answer >= @min_answer::int AND a.answer <= @max_answer::int)
  AND (@search::text = '' OR a.comment ILIKE '%' || @search::text || '%'
       OR (@names::boolean AND coalesce(c.first_name || ' ' || c.last_name, '') ILIKE '%' || @search::text || '%'))
ORDER BY coalesce(a.commented_at, a.updated_at) DESC, a.id DESC
LIMIT @lim OFFSET @off;

-- name: PollCommentsCount :many
SELECT a.answer, count(*)::bigint AS n
FROM poll_answers a
WHERE a.poll_id = @poll_id AND a.kind <> 'test' AND a.comment <> ''
GROUP BY a.answer;

-- name: PollExportAnswers :many
SELECT a.kind, a.answer, a.first_answer, a.changes, a.comment, a.answered_at, a.updated_at,
       coalesce(c.email, '')::text AS email, coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name
FROM poll_answers a
LEFT JOIN contacts c ON c.id = a.contact_id
WHERE a.poll_id = @poll_id AND a.kind <> 'test'
ORDER BY a.answered_at, a.id;

-- name: PollDelivered :one
SELECT coalesce(sum(delivered), 0)::bigint AS delivered, coalesce(sum(sent), 0)::bigint AS sent
FROM campaign_counters
WHERE campaign_id = @campaign_id;

-- name: PollMessage :one
-- The message a signed answer link names: when it was sent and to whom.
SELECT m.id, m.campaign_id, m.contact_id, m.sent_at, m.delivered_at,
       coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name, coalesce(c.email, '')::text AS email,
       coalesce(c.status, '')::text AS contact_status, (c.erased_at IS NOT NULL)::boolean AS erased
FROM messages m
LEFT JOIN contacts c ON c.id = m.contact_id
WHERE m.id = @id;

-- name: PollContact :one
SELECT c.id, c.workspace_id, c.email, c.first_name, c.last_name, c.status, c.tags, c.attributes, (c.erased_at IS NOT NULL)::boolean AS erased
FROM contacts c
WHERE c.id = @id;

-- name: PollSetContactTags :exec
UPDATE contacts SET tags = @tags, updated_at = @at WHERE id = @id;

-- name: PollSetContactField :exec
UPDATE contacts SET attributes = jsonb_set(attributes, ARRAY[@field::text], @value::jsonb, true), updated_at = @at WHERE id = @id;

-- name: PollContactAnswers :many
-- A contact's poll answers, newest first (the profile's timeline and where
-- a field came from).
SELECT a.id, a.answer, a.first_answer, a.changes, a.comment, a.answered_at, a.updated_at,
       p.id AS poll_id, p.kind, p.question, p.answers, p.settings, coalesce(c.name, '')::text AS campaign_name, p.campaign_id
FROM poll_answers a
JOIN polls p ON p.id = a.poll_id
LEFT JOIN campaigns c ON c.id = p.campaign_id
WHERE a.contact_id = @contact_id AND a.kind = 'contact' AND p.workspace_id = @workspace_id
ORDER BY a.updated_at DESC
LIMIT 50;

-- name: SurveysList :many
SELECT s.id, s.number, s.slug, s.name, s.description, s.status, s.questions, s.settings, s.closes_at, s.published_at, s.closed_at,
       s.created_at, s.updated_at, coalesce(u.name, '')::text AS created_by_name, s.created_by,
       (SELECT count(*) FROM survey_responses r WHERE r.survey_id = s.id AND r.kind <> 'test')::bigint AS started,
       (SELECT count(*) FROM survey_responses r WHERE r.survey_id = s.id AND r.kind <> 'test' AND r.completed_at IS NOT NULL)::bigint AS completed
FROM surveys s
LEFT JOIN users u ON u.id = s.created_by
WHERE s.workspace_id = @workspace_id AND s.deleted_at IS NULL
ORDER BY s.created_at DESC;

-- name: SurveyGet :one
SELECT s.id, s.workspace_id, s.number, s.slug, s.name, s.description, s.status, s.questions, s.settings, s.closes_at, s.published_at, s.published_by,
       s.closed_at, s.closed_by, s.created_by, s.updated_by, s.created_at, s.updated_at,
       coalesce(cu.name, '')::text AS created_by_name, coalesce(pu.name, '')::text AS published_by_name, coalesce(xu.name, '')::text AS closed_by_name,
       (SELECT count(*) FROM survey_responses r WHERE r.survey_id = s.id AND r.kind <> 'test')::bigint AS started,
       (SELECT count(*) FROM survey_responses r WHERE r.survey_id = s.id AND r.kind <> 'test' AND r.completed_at IS NOT NULL)::bigint AS completed
FROM surveys s
LEFT JOIN users cu ON cu.id = s.created_by
LEFT JOIN users pu ON pu.id = s.published_by
LEFT JOIN users xu ON xu.id = s.closed_by
WHERE s.workspace_id = @workspace_id AND s.id = @id AND s.deleted_at IS NULL;

-- name: SurveyBySlug :one
SELECT s.id, s.workspace_id, s.number, s.slug, s.name, s.description, s.status, s.questions, s.settings, s.closes_at, s.published_at, s.closed_at
FROM surveys s
WHERE s.workspace_id = @workspace_id AND s.slug = @slug AND s.deleted_at IS NULL;

-- name: SurveyByNumber :one
SELECT s.id, s.workspace_id, s.number, s.slug, s.name, s.description, s.status, s.questions, s.settings, s.closes_at, s.published_at, s.closed_at
FROM surveys s
WHERE s.number = @number AND s.deleted_at IS NULL;

-- name: SurveySlugTaken :one
SELECT EXISTS (SELECT 1 FROM surveys WHERE workspace_id = @workspace_id AND slug = @slug AND id <> @id)::boolean;

-- name: SurveyInsert :exec
INSERT INTO surveys (id, workspace_id, slug, name, description, status, questions, settings, closes_at, created_by, updated_by, created_at, updated_at)
VALUES (@id, @workspace_id, @slug, @name, @description, 'draft', @questions, @settings, sqlc.narg(closes_at), sqlc.narg(created_by), sqlc.narg(created_by), @at, @at);

-- name: SurveyUpdate :exec
UPDATE surveys SET slug = @slug, name = @name, description = @description, questions = @questions, settings = @settings,
    closes_at = sqlc.narg(closes_at), updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE id = @id;

-- name: SurveySetStatus :exec
UPDATE surveys SET status = @status, published_at = coalesce(sqlc.narg(published_at), published_at), published_by = coalesce(sqlc.narg(published_by), published_by),
    closed_at = sqlc.narg(closed_at), closed_by = sqlc.narg(closed_by), closes_at = sqlc.narg(closes_at), updated_at = @at
WHERE id = @id;

-- name: SurveyDelete :exec
UPDATE surveys SET deleted_at = @at, status = 'closed', updated_at = @at WHERE id = @id;

-- name: SurveyResponseOfContact :one
SELECT id, survey_id, workspace_id, contact_id, message_id, kind, test_email, answers, source, started_at, completed_at, updated_at, changes
FROM survey_responses
WHERE survey_id = @survey_id AND contact_id = @contact_id AND kind = 'contact'
ORDER BY updated_at DESC
LIMIT 1;

-- name: SurveyResponseInsert :exec
INSERT INTO survey_responses (id, survey_id, workspace_id, contact_id, message_id, kind, test_email, answers, source, started_at, completed_at, updated_at)
VALUES (@id, @survey_id, @workspace_id, sqlc.narg(contact_id), sqlc.narg(message_id), @kind, @test_email, @answers, @source, @at, sqlc.narg(completed_at), @at);

-- name: SurveyResponseUpdate :exec
UPDATE survey_responses SET answers = @answers, completed_at = coalesce(sqlc.narg(completed_at), completed_at), changes = changes + @changed::int, updated_at = @at
WHERE id = @id;

-- name: SurveyResponses :many
SELECT r.id, r.kind, r.answers, r.source, r.started_at, r.completed_at, r.updated_at, r.contact_id,
       coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name, coalesce(c.email, '')::text AS email
FROM survey_responses r
LEFT JOIN contacts c ON c.id = r.contact_id
WHERE r.survey_id = @survey_id AND r.kind <> 'test' AND r.completed_at IS NOT NULL
ORDER BY r.completed_at DESC, r.id DESC
LIMIT @lim OFFSET @off;

-- name: SurveyAllResponses :many
-- Every response, for the results and the exports.
SELECT r.id, r.kind, r.answers, r.source, r.started_at, r.completed_at, r.updated_at, r.contact_id,
       coalesce(c.first_name, '')::text AS first_name, coalesce(c.last_name, '')::text AS last_name, coalesce(c.email, '')::text AS email
FROM survey_responses r
LEFT JOIN contacts c ON c.id = r.contact_id
WHERE r.survey_id = @survey_id AND r.kind <> 'test'
ORDER BY r.started_at, r.id;

-- name: SurveyContactResponses :many
SELECT r.id, r.answers, r.source, r.started_at, r.completed_at, r.updated_at, s.id AS survey_id, s.name, s.slug, s.questions, s.closes_at, s.settings,
       s.status AS survey_status
FROM survey_responses r
JOIN surveys s ON s.id = r.survey_id
WHERE r.contact_id = @contact_id AND r.kind = 'contact' AND s.workspace_id = @workspace_id AND s.deleted_at IS NULL
ORDER BY r.updated_at DESC
LIMIT 50;

-- name: PollAnswerPrevious :one
-- The answer a changed answer replaced last.
SELECT previous FROM poll_answer_events WHERE answer_id = @answer_id AND kind = 'changed' ORDER BY at DESC, id DESC LIMIT 1;

-- name: SurveyLinkedCampaigns :many
-- Campaigns whose content links to a survey (its link variable or a poll
-- that continues to it).
-- A poll names the survey by its id, however the attribute is escaped.
SELECT c.id, c.name, c.status, c.started_at, c.audience_count, (c.mjml LIKE '%' || @survey_id::text || '%')::boolean AS via_poll
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND (c.mjml ~ @var_pattern::text OR c.mjml LIKE '%' || @survey_id::text || '%')
ORDER BY coalesce(c.started_at, c.created_at) DESC;

-- name: PollCampaignContent :one
SELECT c.id, c.name, c.status, c.mjml, c.html, c.started_at, c.schedule, c.scheduled_at, c.audience
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.id = @id;

-- name: PollAnswerOfContactRead :one
SELECT id, poll_id, workspace_id, contact_id, message_id, kind, test_email, answer, first_answer, changes, comment, commented_at, client, answered_at, updated_at
FROM poll_answers
WHERE poll_id = @poll_id AND contact_id = @contact_id AND kind = 'contact';

-- name: PollAnswerOfTesterRead :one
SELECT id, poll_id, workspace_id, contact_id, message_id, kind, test_email, answer, first_answer, changes, comment, commented_at, client, answered_at, updated_at
FROM poll_answers
WHERE poll_id = @poll_id AND test_email = @test_email AND kind = 'test';

-- name: AutomationPollRefs :many
-- What the trigger "Answered a poll or survey" can point at, by the ref
-- its events carry ("poll:<id>", "survey:<id>").
SELECT ('poll:' || p.id::text)::text AS ref, (coalesce(c.name, '') || ' · ' || p.question)::text AS name
FROM polls p
LEFT JOIN campaigns c ON c.id = p.campaign_id
WHERE p.workspace_id = @workspace_id
UNION ALL
SELECT ('survey:' || s.id::text)::text AS ref, s.name AS name
FROM surveys s
WHERE s.workspace_id = @workspace_id AND s.deleted_at IS NULL;
