from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="HostReputeOutage")



@_attrs_define
class HostReputeOutage:
    """ 
        Attributes:
            message (str): Couldn't reach api.hostrepute.com (timeout after 30 s, 3 attempts)
            since (datetime.datetime | Unset):
     """

    message: str
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        message = self.message

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "message": message,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        message = d.pop("message")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        host_repute_outage = cls(
            message=message,
            since=since,
        )

        return host_repute_outage

