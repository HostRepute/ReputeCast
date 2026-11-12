from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_event_kind import WhatsAppEventKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppEvent")



@_attrs_define
class WhatsAppEvent:
    """ 
        Attributes:
            events (int):
            id (str):
            kind (WhatsAppEventKind):
            received_at (datetime.datetime):
            signature_ok (bool):
            summary (str):
            code (int | Unset):
            error (str | Unset):
            normalized_at (datetime.datetime | Unset):
            number (str | Unset):
            status (str | Unset):
     """

    events: int
    id: str
    kind: WhatsAppEventKind
    received_at: datetime.datetime
    signature_ok: bool
    summary: str
    code: int | Unset = UNSET
    error: str | Unset = UNSET
    normalized_at: datetime.datetime | Unset = UNSET
    number: str | Unset = UNSET
    status: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        events = self.events

        id = self.id

        kind = self.kind.value

        received_at = self.received_at.isoformat()

        signature_ok = self.signature_ok

        summary = self.summary

        code = self.code

        error = self.error

        normalized_at: str | Unset = UNSET
        if not isinstance(self.normalized_at, Unset):
            normalized_at = self.normalized_at.isoformat()

        number = self.number

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "events": events,
            "id": id,
            "kind": kind,
            "receivedAt": received_at,
            "signatureOk": signature_ok,
            "summary": summary,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if error is not UNSET:
            field_dict["error"] = error
        if normalized_at is not UNSET:
            field_dict["normalizedAt"] = normalized_at
        if number is not UNSET:
            field_dict["number"] = number
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        events = d.pop("events")

        id = d.pop("id")

        kind = WhatsAppEventKind(d.pop("kind"))




        received_at = datetime.datetime.fromisoformat(d.pop("receivedAt"))




        signature_ok = d.pop("signatureOk")

        summary = d.pop("summary")

        code = d.pop("code", UNSET)

        error = d.pop("error", UNSET)

        _normalized_at = d.pop("normalizedAt", UNSET)
        normalized_at: datetime.datetime | Unset
        if isinstance(_normalized_at,  Unset):
            normalized_at = UNSET
        else:
            normalized_at = datetime.datetime.fromisoformat(_normalized_at)




        number = d.pop("number", UNSET)

        status = d.pop("status", UNSET)

        whats_app_event = cls(
            events=events,
            id=id,
            kind=kind,
            received_at=received_at,
            signature_ok=signature_ok,
            summary=summary,
            code=code,
            error=error,
            normalized_at=normalized_at,
            number=number,
            status=status,
        )

        return whats_app_event

