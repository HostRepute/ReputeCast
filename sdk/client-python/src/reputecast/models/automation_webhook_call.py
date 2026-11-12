from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AutomationWebhookCall")



@_attrs_define
class AutomationWebhookCall:
    """ 
        Attributes:
            attempt (int):
            called_at (datetime.datetime):
            duration_ms (int):
            id (str):
            status_code (int):
            url (str):
            version (int):
            error (str | Unset):
     """

    attempt: int
    called_at: datetime.datetime
    duration_ms: int
    id: str
    status_code: int
    url: str
    version: int
    error: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        attempt = self.attempt

        called_at = self.called_at.isoformat()

        duration_ms = self.duration_ms

        id = self.id

        status_code = self.status_code

        url = self.url

        version = self.version

        error = self.error


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attempt": attempt,
            "calledAt": called_at,
            "durationMs": duration_ms,
            "id": id,
            "statusCode": status_code,
            "url": url,
            "version": version,
        })
        if error is not UNSET:
            field_dict["error"] = error

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attempt = d.pop("attempt")

        called_at = datetime.datetime.fromisoformat(d.pop("calledAt"))




        duration_ms = d.pop("durationMs")

        id = d.pop("id")

        status_code = d.pop("statusCode")

        url = d.pop("url")

        version = d.pop("version")

        error = d.pop("error", UNSET)

        automation_webhook_call = cls(
            attempt=attempt,
            called_at=called_at,
            duration_ms=duration_ms,
            id=id,
            status_code=status_code,
            url=url,
            version=version,
            error=error,
        )

        return automation_webhook_call

