from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.push_rotation_row import PushRotationRow





T = TypeVar("T", bound="PushRotation")



@_attrs_define
class PushRotation:
    """ 
        Attributes:
            confirm (str):
            scheduled (list[str]):
            sites (list[PushRotationRow]):
            subscriptions (int):
     """

    confirm: str
    scheduled: list[str]
    sites: list[PushRotationRow]
    subscriptions: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_rotation_row import PushRotationRow # noqa: PLC0415
        confirm = self.confirm

        scheduled = self.scheduled



        sites = []
        for sites_item_data in self.sites:
            sites_item = sites_item_data.to_dict()
            sites.append(sites_item)



        subscriptions = self.subscriptions


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirm": confirm,
            "scheduled": scheduled,
            "sites": sites,
            "subscriptions": subscriptions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_rotation_row import PushRotationRow # noqa: PLC0415
        d = dict(src_dict)
        confirm = d.pop("confirm")

        scheduled = cast(list[str], d.pop("scheduled"))


        sites = []
        _sites = d.pop("sites")
        for sites_item_data in (_sites):
            sites_item = PushRotationRow.from_dict(sites_item_data)



            sites.append(sites_item)


        subscriptions = d.pop("subscriptions")

        push_rotation = cls(
            confirm=confirm,
            scheduled=scheduled,
            sites=sites,
            subscriptions=subscriptions,
        )

        return push_rotation

