"""End-to-end check of the package against a running ReputeCast instance with the Acme demo.

internal/sdktest runs it against a fresh one: REPUTECAST_URL, REPUTECAST_API_KEY
(subscribers:view, subscribers:create, subscribers:edit and
transactional:send, without team:view) and REPUTECAST_WORKSPACE.
"""

from __future__ import annotations

import os

from reputecast import ReputeCast, ReputeCastError, unwrap
from reputecast.api.audience import list_contacts, list_lists
from reputecast.api.contacts import upsert_contact
from reputecast.api.team import list_members
from reputecast.api.transactional_messages import send_transactional_message
from reputecast.models import ContactUpsertInput, TransactionalSend


def main() -> None:
    rc = ReputeCast(api_key=os.environ["REPUTECAST_API_KEY"], base_url=os.environ["REPUTECAST_URL"])
    workspace = os.environ.get("REPUTECAST_WORKSPACE", "acme-newsletter")

    # Lists.
    lists = unwrap(list_lists.sync_detailed(client=rc, workspace=workspace))
    assert lists.items, "the demo has lists"
    digest = next((item for item in lists.items if item.name == "Weekly Digest"), lists.items[0])

    # Upsert a contact, twice: created, then updated.
    email = "sdk-e2e-python@example.com"
    body = {"email": email, "firstName": "Pia", "lists": [digest.id], "tags": ["sdk-e2e"], "consent": "single"}
    first = unwrap(upsert_contact.sync_detailed(client=rc, workspace=workspace, body=ContactUpsertInput.from_dict(body)))
    assert first.created
    again = unwrap(upsert_contact.sync_detailed(client=rc, workspace=workspace, body=ContactUpsertInput.from_dict({**body, "lastName": "Thon"})))
    assert not again.created and again.contact_id == first.contact_id

    # Page through contacts with the cursor: three pages of 25.
    seen: set[str] = set()
    cursor: str | None = None
    for _ in range(3):
        kwargs = {"cursor": cursor} if cursor else {}
        page = unwrap(list_contacts.sync_detailed(client=rc, workspace=workspace, page_size=25, **kwargs))
        for contact in page.items:
            assert contact.id not in seen, f"contact {contact.id} came twice"
            seen.add(contact.id)
        assert page.next_cursor, "there is another page"
        cursor = page.next_cursor
    assert len(seen) == 75, f"paged through {len(seen)} contacts"
    found = unwrap(list_contacts.sync_detailed(client=rc, workspace=workspace, q=email))
    assert found.items[0].id == first.contact_id

    # A transactional message, with an Idempotency-Key: a repeat sends nothing new.
    user = {"id": "u_42", "email": "sam.rivera@outlook.com", "name": "Sam Rivera", "first_name": "Sam"}
    token = {"id": "t_7"}

    def send():
        return send_transactional_message.sync_detailed(
            client=rc,
            workspace=workspace,
            idempotency_key=f"pwreset-{user['id']}-{token['id']}",
            body=TransactionalSend.from_dict(
                {
                    "template": "password-reset",
                    "to": {"email": user["email"], "name": user["name"]},
                    "data": {"first_name": user["first_name"], "reset_url": f"https://app.acme.io/reset/{token['id']}", "expires_in": "30 minutes"},
                }
            ),
        )

    msg = unwrap(send())
    assert msg.id and msg.status
    assert unwrap(send()).id == msg.id

    # A missing permission is a ReputeCastError with the problem's fields.
    try:
        unwrap(list_members.sync_detailed(client=rc, workspace=workspace))
    except ReputeCastError as err:
        assert (err.status, err.code, err.permission) == (403, "permission_denied", "team:view"), repr(err)
        assert err.detail
    else:
        raise AssertionError("listing members needs team:view")

    print(f"ok python: {len(lists.items)} lists, {len(seen)} contacts paged, message {msg.status}, members answered 403 permission_denied")


if __name__ == "__main__":
    main()
