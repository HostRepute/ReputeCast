from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationAttempt")



@_attrs_define
class IntegrationAttempt:
    """ 
        Attributes:
            attempt (int):
            event (str):
            max_attempts (int):
            queued (int):
            at (datetime.datetime | Unset):
            status (int | Unset):
            status_text (str | Unset):
     """

    attempt: int
    event: str
    max_attempts: int
    queued: int
    at: datetime.datetime | Unset = UNSET
    status: int | Unset = UNSET
    status_text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        attempt = self.attempt

        event = self.event

        max_attempts = self.max_attempts

        queued = self.queued

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        status = self.status

        status_text = self.status_text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attempt": attempt,
            "event": event,
            "maxAttempts": max_attempts,
            "queued": queued,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if status is not UNSET:
            field_dict["status"] = status
        if status_text is not UNSET:
            field_dict["statusText"] = status_text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attempt = d.pop("attempt")

        event = d.pop("event")

        max_attempts = d.pop("maxAttempts")

        queued = d.pop("queued")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        status = d.pop("status", UNSET)

        status_text = d.pop("statusText", UNSET)

        integration_attempt = cls(
            attempt=attempt,
            event=event,
            max_attempts=max_attempts,
            queued=queued,
            at=at,
            status=status,
            status_text=status_text,
        )

        return integration_attempt

