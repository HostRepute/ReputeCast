from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_event_tone import LiveEventTone
from typing import cast
import datetime






T = TypeVar("T", bound="LiveEvent")



@_attrs_define
class LiveEvent:
    """ 
        Attributes:
            address (str):
            at (datetime.datetime):
            detail (str):
            id (str):
            kind (str):
            label (str):
            short (str):
            tone (LiveEventTone):
     """

    address: str
    at: datetime.datetime
    detail: str
    id: str
    kind: str
    label: str
    short: str
    tone: LiveEventTone





    def to_dict(self) -> dict[str, Any]:
        address = self.address

        at = self.at.isoformat()

        detail = self.detail

        id = self.id

        kind = self.kind

        label = self.label

        short = self.short

        tone = self.tone.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "at": at,
            "detail": detail,
            "id": id,
            "kind": kind,
            "label": label,
            "short": short,
            "tone": tone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        address = d.pop("address")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        detail = d.pop("detail")

        id = d.pop("id")

        kind = d.pop("kind")

        label = d.pop("label")

        short = d.pop("short")

        tone = LiveEventTone(d.pop("tone"))




        live_event = cls(
            address=address,
            at=at,
            detail=detail,
            id=id,
            kind=kind,
            label=label,
            short=short,
            tone=tone,
        )

        return live_event

