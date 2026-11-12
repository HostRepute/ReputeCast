from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_message_view_status import SmsMessageViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_message_event import SmsMessageEvent





T = TypeVar("T", bound="SmsMessageView")



@_attrs_define
class SmsMessageView:
    """ 
        Attributes:
            body (str):
            characters (int):
            created_at (datetime.datetime):
            encoding (str):
            events (list[SmsMessageEvent]):
            id (str):
            kind (str):
            provider (str):
            recipient (str):
            segments (int):
            sender (str):
            status (SmsMessageViewStatus):
            delivered_at (datetime.datetime | Unset):
            provider_message_id (str | Unset):
            reason (str | Unset):
            seconds (float | Unset):
            sent_at (datetime.datetime | Unset):
     """

    body: str
    characters: int
    created_at: datetime.datetime
    encoding: str
    events: list[SmsMessageEvent]
    id: str
    kind: str
    provider: str
    recipient: str
    segments: int
    sender: str
    status: SmsMessageViewStatus
    delivered_at: datetime.datetime | Unset = UNSET
    provider_message_id: str | Unset = UNSET
    reason: str | Unset = UNSET
    seconds: float | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_message_event import SmsMessageEvent # noqa: PLC0415
        body = self.body

        characters = self.characters

        created_at = self.created_at.isoformat()

        encoding = self.encoding

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        id = self.id

        kind = self.kind

        provider = self.provider

        recipient = self.recipient

        segments = self.segments

        sender = self.sender

        status = self.status.value

        delivered_at: str | Unset = UNSET
        if not isinstance(self.delivered_at, Unset):
            delivered_at = self.delivered_at.isoformat()

        provider_message_id = self.provider_message_id

        reason = self.reason

        seconds = self.seconds

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "characters": characters,
            "createdAt": created_at,
            "encoding": encoding,
            "events": events,
            "id": id,
            "kind": kind,
            "provider": provider,
            "recipient": recipient,
            "segments": segments,
            "sender": sender,
            "status": status,
        })
        if delivered_at is not UNSET:
            field_dict["deliveredAt"] = delivered_at
        if provider_message_id is not UNSET:
            field_dict["providerMessageId"] = provider_message_id
        if reason is not UNSET:
            field_dict["reason"] = reason
        if seconds is not UNSET:
            field_dict["seconds"] = seconds
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_message_event import SmsMessageEvent # noqa: PLC0415
        d = dict(src_dict)
        body = d.pop("body")

        characters = d.pop("characters")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        encoding = d.pop("encoding")

        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = SmsMessageEvent.from_dict(events_item_data)



            events.append(events_item)


        id = d.pop("id")

        kind = d.pop("kind")

        provider = d.pop("provider")

        recipient = d.pop("recipient")

        segments = d.pop("segments")

        sender = d.pop("sender")

        status = SmsMessageViewStatus(d.pop("status"))




        _delivered_at = d.pop("deliveredAt", UNSET)
        delivered_at: datetime.datetime | Unset
        if isinstance(_delivered_at,  Unset):
            delivered_at = UNSET
        else:
            delivered_at = datetime.datetime.fromisoformat(_delivered_at)




        provider_message_id = d.pop("providerMessageId", UNSET)

        reason = d.pop("reason", UNSET)

        seconds = d.pop("seconds", UNSET)

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        sms_message_view = cls(
            body=body,
            characters=characters,
            created_at=created_at,
            encoding=encoding,
            events=events,
            id=id,
            kind=kind,
            provider=provider,
            recipient=recipient,
            segments=segments,
            sender=sender,
            status=status,
            delivered_at=delivered_at,
            provider_message_id=provider_message_id,
            reason=reason,
            seconds=seconds,
            sent_at=sent_at,
        )

        return sms_message_view

