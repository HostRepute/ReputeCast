from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="APIKeyRejections")



@_attrs_define
class APIKeyRejections:
    """ 
        Attributes:
            calls (int): In the last 7 days
            client (str | Unset): Client label of that address
            ip (str | Unset): Busiest client address
            since (datetime.datetime | Unset): First rejected call
     """

    calls: int
    client: str | Unset = UNSET
    ip: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        calls = self.calls

        client = self.client

        ip = self.ip

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "calls": calls,
        })
        if client is not UNSET:
            field_dict["client"] = client
        if ip is not UNSET:
            field_dict["ip"] = ip
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        calls = d.pop("calls")

        client = d.pop("client", UNSET)

        ip = d.pop("ip", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        api_key_rejections = cls(
            calls=calls,
            client=client,
            ip=ip,
            since=since,
        )

        return api_key_rejections

