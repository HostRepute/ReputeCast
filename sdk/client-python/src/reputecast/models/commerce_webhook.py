from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_webhook_status import CommerceWebhookStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceWebhook")



@_attrs_define
class CommerceWebhook:
    """ 
        Attributes:
            status (CommerceWebhookStatus):
            topic (str):
            error (str | Unset):
            last_event_at (datetime.datetime | Unset):
            missing_since (datetime.datetime | Unset):
     """

    status: CommerceWebhookStatus
    topic: str
    error: str | Unset = UNSET
    last_event_at: datetime.datetime | Unset = UNSET
    missing_since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        topic = self.topic

        error = self.error

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        missing_since: str | Unset = UNSET
        if not isinstance(self.missing_since, Unset):
            missing_since = self.missing_since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
            "topic": topic,
        })
        if error is not UNSET:
            field_dict["error"] = error
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if missing_since is not UNSET:
            field_dict["missingSince"] = missing_since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = CommerceWebhookStatus(d.pop("status"))




        topic = d.pop("topic")

        error = d.pop("error", UNSET)

        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        _missing_since = d.pop("missingSince", UNSET)
        missing_since: datetime.datetime | Unset
        if isinstance(_missing_since,  Unset):
            missing_since = UNSET
        else:
            missing_since = datetime.datetime.fromisoformat(_missing_since)




        commerce_webhook = cls(
            status=status,
            topic=topic,
            error=error,
            last_event_at=last_event_at,
            missing_since=missing_since,
        )

        return commerce_webhook

