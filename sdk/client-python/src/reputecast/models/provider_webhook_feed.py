from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderWebhookFeed")



@_attrs_define
class ProviderWebhookFeed:
    """ 
        Attributes:
            kind (str):
            last_event_at (datetime.datetime | Unset):
            silent_since (datetime.datetime | Unset):
     """

    kind: str
    last_event_at: datetime.datetime | Unset = UNSET
    silent_since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        silent_since: str | Unset = UNSET
        if not isinstance(self.silent_since, Unset):
            silent_since = self.silent_since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if silent_since is not UNSET:
            field_dict["silentSince"] = silent_since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        _silent_since = d.pop("silentSince", UNSET)
        silent_since: datetime.datetime | Unset
        if isinstance(_silent_since,  Unset):
            silent_since = UNSET
        else:
            silent_since = datetime.datetime.fromisoformat(_silent_since)




        provider_webhook_feed = cls(
            kind=kind,
            last_event_at=last_event_at,
            silent_since=silent_since,
        )

        return provider_webhook_feed

