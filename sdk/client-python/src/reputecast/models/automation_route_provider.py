from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AutomationRouteProvider")



@_attrs_define
class AutomationRouteProvider:
    """ 
        Attributes:
            adapter (str):
            id (str):
            name (str):  Example: Amazon SES.
            short (str):  Example: SES.
            status (str): healthy, degraded, down, sandbox, warming_up or paused
            detail (str | Unset):
            down_reason (str | Unset):
            down_since (datetime.datetime | Unset):
            last_error_status (int | Unset):
            weight (int | Unset):
     """

    adapter: str
    id: str
    name: str
    short: str
    status: str
    detail: str | Unset = UNSET
    down_reason: str | Unset = UNSET
    down_since: datetime.datetime | Unset = UNSET
    last_error_status: int | Unset = UNSET
    weight: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        id = self.id

        name = self.name

        short = self.short

        status = self.status

        detail = self.detail

        down_reason = self.down_reason

        down_since: str | Unset = UNSET
        if not isinstance(self.down_since, Unset):
            down_since = self.down_since.isoformat()

        last_error_status = self.last_error_status

        weight = self.weight


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "id": id,
            "name": name,
            "short": short,
            "status": status,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if down_reason is not UNSET:
            field_dict["downReason"] = down_reason
        if down_since is not UNSET:
            field_dict["downSince"] = down_since
        if last_error_status is not UNSET:
            field_dict["lastErrorStatus"] = last_error_status
        if weight is not UNSET:
            field_dict["weight"] = weight

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        id = d.pop("id")

        name = d.pop("name")

        short = d.pop("short")

        status = d.pop("status")

        detail = d.pop("detail", UNSET)

        down_reason = d.pop("downReason", UNSET)

        _down_since = d.pop("downSince", UNSET)
        down_since: datetime.datetime | Unset
        if isinstance(_down_since,  Unset):
            down_since = UNSET
        else:
            down_since = datetime.datetime.fromisoformat(_down_since)




        last_error_status = d.pop("lastErrorStatus", UNSET)

        weight = d.pop("weight", UNSET)

        automation_route_provider = cls(
            adapter=adapter,
            id=id,
            name=name,
            short=short,
            status=status,
            detail=detail,
            down_reason=down_reason,
            down_since=down_since,
            last_error_status=last_error_status,
            weight=weight,
        )

        return automation_route_provider

