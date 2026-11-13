from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_comment_held_reason import WebsiteCommentHeldReason
from ..models.website_comment_status import WebsiteCommentStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_comment_reply import WebsiteCommentReply





T = TypeVar("T", bound="WebsiteComment")



@_attrs_define
class WebsiteComment:
    """ 
        Attributes:
            blocked (bool):
            body (str):
            created_at (datetime.datetime):
            id (str):
            links (int):
            name (str):
            pinned (bool):
            post_id (str):
            post_title (str):
            replies (int):
            status (WebsiteCommentStatus):
            team (bool):
            url (str):
            by (str | Unset):
            contact_id (str | Unset):
            edited_at (datetime.datetime | Unset):
            email (str | Unset):
            first_reply (WebsiteCommentReply | Unset):
            held_reason (WebsiteCommentHeldReason | Unset):
            moderated_at (datetime.datetime | Unset):
            moderated_by (str | Unset):
            parent_id (str | Unset):
            parent_name (str | Unset):
            published_at (datetime.datetime | Unset):
     """

    blocked: bool
    body: str
    created_at: datetime.datetime
    id: str
    links: int
    name: str
    pinned: bool
    post_id: str
    post_title: str
    replies: int
    status: WebsiteCommentStatus
    team: bool
    url: str
    by: str | Unset = UNSET
    contact_id: str | Unset = UNSET
    edited_at: datetime.datetime | Unset = UNSET
    email: str | Unset = UNSET
    first_reply: WebsiteCommentReply | Unset = UNSET
    held_reason: WebsiteCommentHeldReason | Unset = UNSET
    moderated_at: datetime.datetime | Unset = UNSET
    moderated_by: str | Unset = UNSET
    parent_id: str | Unset = UNSET
    parent_name: str | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_comment_reply import WebsiteCommentReply # noqa: PLC0415
        blocked = self.blocked

        body = self.body

        created_at = self.created_at.isoformat()

        id = self.id

        links = self.links

        name = self.name

        pinned = self.pinned

        post_id = self.post_id

        post_title = self.post_title

        replies = self.replies

        status = self.status.value

        team = self.team

        url = self.url

        by = self.by

        contact_id = self.contact_id

        edited_at: str | Unset = UNSET
        if not isinstance(self.edited_at, Unset):
            edited_at = self.edited_at.isoformat()

        email = self.email

        first_reply: dict[str, Any] | Unset = UNSET
        if not isinstance(self.first_reply, Unset):
            first_reply = self.first_reply.to_dict()

        held_reason: str | Unset = UNSET
        if not isinstance(self.held_reason, Unset):
            held_reason = self.held_reason.value


        moderated_at: str | Unset = UNSET
        if not isinstance(self.moderated_at, Unset):
            moderated_at = self.moderated_at.isoformat()

        moderated_by = self.moderated_by

        parent_id = self.parent_id

        parent_name = self.parent_name

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocked": blocked,
            "body": body,
            "createdAt": created_at,
            "id": id,
            "links": links,
            "name": name,
            "pinned": pinned,
            "postId": post_id,
            "postTitle": post_title,
            "replies": replies,
            "status": status,
            "team": team,
            "url": url,
        })
        if by is not UNSET:
            field_dict["by"] = by
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if edited_at is not UNSET:
            field_dict["editedAt"] = edited_at
        if email is not UNSET:
            field_dict["email"] = email
        if first_reply is not UNSET:
            field_dict["firstReply"] = first_reply
        if held_reason is not UNSET:
            field_dict["heldReason"] = held_reason
        if moderated_at is not UNSET:
            field_dict["moderatedAt"] = moderated_at
        if moderated_by is not UNSET:
            field_dict["moderatedBy"] = moderated_by
        if parent_id is not UNSET:
            field_dict["parentId"] = parent_id
        if parent_name is not UNSET:
            field_dict["parentName"] = parent_name
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_comment_reply import WebsiteCommentReply # noqa: PLC0415
        d = dict(src_dict)
        blocked = d.pop("blocked")

        body = d.pop("body")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        id = d.pop("id")

        links = d.pop("links")

        name = d.pop("name")

        pinned = d.pop("pinned")

        post_id = d.pop("postId")

        post_title = d.pop("postTitle")

        replies = d.pop("replies")

        status = WebsiteCommentStatus(d.pop("status"))




        team = d.pop("team")

        url = d.pop("url")

        by = d.pop("by", UNSET)

        contact_id = d.pop("contactId", UNSET)

        _edited_at = d.pop("editedAt", UNSET)
        edited_at: datetime.datetime | Unset
        if isinstance(_edited_at,  Unset):
            edited_at = UNSET
        else:
            edited_at = datetime.datetime.fromisoformat(_edited_at)




        email = d.pop("email", UNSET)

        _first_reply = d.pop("firstReply", UNSET)
        first_reply: WebsiteCommentReply | Unset
        if isinstance(_first_reply,  Unset):
            first_reply = UNSET
        else:
            first_reply = WebsiteCommentReply.from_dict(_first_reply)




        _held_reason = d.pop("heldReason", UNSET)
        held_reason: WebsiteCommentHeldReason | Unset
        if isinstance(_held_reason,  Unset):
            held_reason = UNSET
        else:
            held_reason = WebsiteCommentHeldReason(_held_reason)




        _moderated_at = d.pop("moderatedAt", UNSET)
        moderated_at: datetime.datetime | Unset
        if isinstance(_moderated_at,  Unset):
            moderated_at = UNSET
        else:
            moderated_at = datetime.datetime.fromisoformat(_moderated_at)




        moderated_by = d.pop("moderatedBy", UNSET)

        parent_id = d.pop("parentId", UNSET)

        parent_name = d.pop("parentName", UNSET)

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        website_comment = cls(
            blocked=blocked,
            body=body,
            created_at=created_at,
            id=id,
            links=links,
            name=name,
            pinned=pinned,
            post_id=post_id,
            post_title=post_title,
            replies=replies,
            status=status,
            team=team,
            url=url,
            by=by,
            contact_id=contact_id,
            edited_at=edited_at,
            email=email,
            first_reply=first_reply,
            held_reason=held_reason,
            moderated_at=moderated_at,
            moderated_by=moderated_by,
            parent_id=parent_id,
            parent_name=parent_name,
            published_at=published_at,
        )

        return website_comment

