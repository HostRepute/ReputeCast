from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_history_item_kind import DigestHistoryItemKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DigestHistoryItem")



@_attrs_define
class DigestHistoryItem:
    """ 
        Attributes:
            at (datetime.datetime):
            kind (DigestHistoryItemKind):
            actor (str | Unset):
            approved_at (datetime.datetime | Unset):
            approved_by (str | Unset):
            built_at (datetime.datetime | Unset):
            built_by (str | Unset):
            built_mode (str | Unset):
            category (str | Unset):
            draft_list (str | Unset):
            left_out_by (str | Unset):
            mode (str | Unset):
            note (str | Unset):
            number (int | Unset):
            owner (str | Unset):
            pause_reason (str | Unset):
            published (datetime.datetime | Unset):
            recipients (int | Unset):
            send_at (datetime.datetime | Unset):
            skip_reason (str | Unset):
            title (str | Unset):
            via (str | Unset):
     """

    at: datetime.datetime
    kind: DigestHistoryItemKind
    actor: str | Unset = UNSET
    approved_at: datetime.datetime | Unset = UNSET
    approved_by: str | Unset = UNSET
    built_at: datetime.datetime | Unset = UNSET
    built_by: str | Unset = UNSET
    built_mode: str | Unset = UNSET
    category: str | Unset = UNSET
    draft_list: str | Unset = UNSET
    left_out_by: str | Unset = UNSET
    mode: str | Unset = UNSET
    note: str | Unset = UNSET
    number: int | Unset = UNSET
    owner: str | Unset = UNSET
    pause_reason: str | Unset = UNSET
    published: datetime.datetime | Unset = UNSET
    recipients: int | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    skip_reason: str | Unset = UNSET
    title: str | Unset = UNSET
    via: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        kind = self.kind.value

        actor = self.actor

        approved_at: str | Unset = UNSET
        if not isinstance(self.approved_at, Unset):
            approved_at = self.approved_at.isoformat()

        approved_by = self.approved_by

        built_at: str | Unset = UNSET
        if not isinstance(self.built_at, Unset):
            built_at = self.built_at.isoformat()

        built_by = self.built_by

        built_mode = self.built_mode

        category = self.category

        draft_list = self.draft_list

        left_out_by = self.left_out_by

        mode = self.mode

        note = self.note

        number = self.number

        owner = self.owner

        pause_reason = self.pause_reason

        published: str | Unset = UNSET
        if not isinstance(self.published, Unset):
            published = self.published.isoformat()

        recipients = self.recipients

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        skip_reason = self.skip_reason

        title = self.title

        via = self.via


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "kind": kind,
        })
        if actor is not UNSET:
            field_dict["actor"] = actor
        if approved_at is not UNSET:
            field_dict["approvedAt"] = approved_at
        if approved_by is not UNSET:
            field_dict["approvedBy"] = approved_by
        if built_at is not UNSET:
            field_dict["builtAt"] = built_at
        if built_by is not UNSET:
            field_dict["builtBy"] = built_by
        if built_mode is not UNSET:
            field_dict["builtMode"] = built_mode
        if category is not UNSET:
            field_dict["category"] = category
        if draft_list is not UNSET:
            field_dict["draftList"] = draft_list
        if left_out_by is not UNSET:
            field_dict["leftOutBy"] = left_out_by
        if mode is not UNSET:
            field_dict["mode"] = mode
        if note is not UNSET:
            field_dict["note"] = note
        if number is not UNSET:
            field_dict["number"] = number
        if owner is not UNSET:
            field_dict["owner"] = owner
        if pause_reason is not UNSET:
            field_dict["pauseReason"] = pause_reason
        if published is not UNSET:
            field_dict["published"] = published
        if recipients is not UNSET:
            field_dict["recipients"] = recipients
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at
        if skip_reason is not UNSET:
            field_dict["skipReason"] = skip_reason
        if title is not UNSET:
            field_dict["title"] = title
        if via is not UNSET:
            field_dict["via"] = via

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        kind = DigestHistoryItemKind(d.pop("kind"))




        actor = d.pop("actor", UNSET)

        _approved_at = d.pop("approvedAt", UNSET)
        approved_at: datetime.datetime | Unset
        if isinstance(_approved_at,  Unset):
            approved_at = UNSET
        else:
            approved_at = datetime.datetime.fromisoformat(_approved_at)




        approved_by = d.pop("approvedBy", UNSET)

        _built_at = d.pop("builtAt", UNSET)
        built_at: datetime.datetime | Unset
        if isinstance(_built_at,  Unset):
            built_at = UNSET
        else:
            built_at = datetime.datetime.fromisoformat(_built_at)




        built_by = d.pop("builtBy", UNSET)

        built_mode = d.pop("builtMode", UNSET)

        category = d.pop("category", UNSET)

        draft_list = d.pop("draftList", UNSET)

        left_out_by = d.pop("leftOutBy", UNSET)

        mode = d.pop("mode", UNSET)

        note = d.pop("note", UNSET)

        number = d.pop("number", UNSET)

        owner = d.pop("owner", UNSET)

        pause_reason = d.pop("pauseReason", UNSET)

        _published = d.pop("published", UNSET)
        published: datetime.datetime | Unset
        if isinstance(_published,  Unset):
            published = UNSET
        else:
            published = datetime.datetime.fromisoformat(_published)




        recipients = d.pop("recipients", UNSET)

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        skip_reason = d.pop("skipReason", UNSET)

        title = d.pop("title", UNSET)

        via = d.pop("via", UNSET)

        digest_history_item = cls(
            at=at,
            kind=kind,
            actor=actor,
            approved_at=approved_at,
            approved_by=approved_by,
            built_at=built_at,
            built_by=built_by,
            built_mode=built_mode,
            category=category,
            draft_list=draft_list,
            left_out_by=left_out_by,
            mode=mode,
            note=note,
            number=number,
            owner=owner,
            pause_reason=pause_reason,
            published=published,
            recipients=recipients,
            send_at=send_at,
            skip_reason=skip_reason,
            title=title,
            via=via,
        )

        return digest_history_item

