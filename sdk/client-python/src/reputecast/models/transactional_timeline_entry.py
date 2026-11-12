from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_timeline_entry_kind import TransactionalTimelineEntryKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalTimelineEntry")



@_attrs_define
class TransactionalTimelineEntry:
    """ 
        Attributes:
            at (datetime.datetime):
            kind (TransactionalTimelineEntryKind):
            code (str | Unset):
            detail (str | Unset):
            down (bool | Unset):
            event (str | Unset):
            machine (bool | Unset):
            provider (str | Unset):
            provider_message_id (str | Unset):
            state (str | Unset):
            status (int | Unset):
            to (str | Unset):
            url (str | Unset):
     """

    at: datetime.datetime
    kind: TransactionalTimelineEntryKind
    code: str | Unset = UNSET
    detail: str | Unset = UNSET
    down: bool | Unset = UNSET
    event: str | Unset = UNSET
    machine: bool | Unset = UNSET
    provider: str | Unset = UNSET
    provider_message_id: str | Unset = UNSET
    state: str | Unset = UNSET
    status: int | Unset = UNSET
    to: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        kind = self.kind.value

        code = self.code

        detail = self.detail

        down = self.down

        event = self.event

        machine = self.machine

        provider = self.provider

        provider_message_id = self.provider_message_id

        state = self.state

        status = self.status

        to = self.to

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "kind": kind,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if detail is not UNSET:
            field_dict["detail"] = detail
        if down is not UNSET:
            field_dict["down"] = down
        if event is not UNSET:
            field_dict["event"] = event
        if machine is not UNSET:
            field_dict["machine"] = machine
        if provider is not UNSET:
            field_dict["provider"] = provider
        if provider_message_id is not UNSET:
            field_dict["providerMessageId"] = provider_message_id
        if state is not UNSET:
            field_dict["state"] = state
        if status is not UNSET:
            field_dict["status"] = status
        if to is not UNSET:
            field_dict["to"] = to
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        kind = TransactionalTimelineEntryKind(d.pop("kind"))




        code = d.pop("code", UNSET)

        detail = d.pop("detail", UNSET)

        down = d.pop("down", UNSET)

        event = d.pop("event", UNSET)

        machine = d.pop("machine", UNSET)

        provider = d.pop("provider", UNSET)

        provider_message_id = d.pop("providerMessageId", UNSET)

        state = d.pop("state", UNSET)

        status = d.pop("status", UNSET)

        to = d.pop("to", UNSET)

        url = d.pop("url", UNSET)

        transactional_timeline_entry = cls(
            at=at,
            kind=kind,
            code=code,
            detail=detail,
            down=down,
            event=event,
            machine=machine,
            provider=provider,
            provider_message_id=provider_message_id,
            state=state,
            status=status,
            to=to,
            url=url,
        )

        return transactional_timeline_entry

