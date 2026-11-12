from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AuditStreamStatus")



@_attrs_define
class AuditStreamStatus:
    """ 
        Attributes:
            delivered (int):
            enabled (bool):
            has_secret (bool):
            failing_since (datetime.datetime | Unset):
            host (str | Unset):
            last_error (str | Unset):
            last_ok_at (datetime.datetime | Unset):
            url (str | Unset):
     """

    delivered: int
    enabled: bool
    has_secret: bool
    failing_since: datetime.datetime | Unset = UNSET
    host: str | Unset = UNSET
    last_error: str | Unset = UNSET
    last_ok_at: datetime.datetime | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        delivered = self.delivered

        enabled = self.enabled

        has_secret = self.has_secret

        failing_since: str | Unset = UNSET
        if not isinstance(self.failing_since, Unset):
            failing_since = self.failing_since.isoformat()

        host = self.host

        last_error = self.last_error

        last_ok_at: str | Unset = UNSET
        if not isinstance(self.last_ok_at, Unset):
            last_ok_at = self.last_ok_at.isoformat()

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "delivered": delivered,
            "enabled": enabled,
            "hasSecret": has_secret,
        })
        if failing_since is not UNSET:
            field_dict["failingSince"] = failing_since
        if host is not UNSET:
            field_dict["host"] = host
        if last_error is not UNSET:
            field_dict["lastError"] = last_error
        if last_ok_at is not UNSET:
            field_dict["lastOkAt"] = last_ok_at
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        delivered = d.pop("delivered")

        enabled = d.pop("enabled")

        has_secret = d.pop("hasSecret")

        _failing_since = d.pop("failingSince", UNSET)
        failing_since: datetime.datetime | Unset
        if isinstance(_failing_since,  Unset):
            failing_since = UNSET
        else:
            failing_since = datetime.datetime.fromisoformat(_failing_since)




        host = d.pop("host", UNSET)

        last_error = d.pop("lastError", UNSET)

        _last_ok_at = d.pop("lastOkAt", UNSET)
        last_ok_at: datetime.datetime | Unset
        if isinstance(_last_ok_at,  Unset):
            last_ok_at = UNSET
        else:
            last_ok_at = datetime.datetime.fromisoformat(_last_ok_at)




        url = d.pop("url", UNSET)

        audit_stream_status = cls(
            delivered=delivered,
            enabled=enabled,
            has_secret=has_secret,
            failing_since=failing_since,
            host=host,
            last_error=last_error,
            last_ok_at=last_ok_at,
            url=url,
        )

        return audit_stream_status

