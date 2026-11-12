from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="OverviewListGrowth")



@_attrs_define
class OverviewListGrowth:
    """ 
        Attributes:
            contacts (int):
            id (str):
            name (str):
            net (int):
     """

    contacts: int
    id: str
    name: str
    net: int





    def to_dict(self) -> dict[str, Any]:
        contacts = self.contacts

        id = self.id

        name = self.name

        net = self.net


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "id": id,
            "name": name,
            "net": net,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contacts = d.pop("contacts")

        id = d.pop("id")

        name = d.pop("name")

        net = d.pop("net")

        overview_list_growth = cls(
            contacts=contacts,
            id=id,
            name=name,
            net=net,
        )

        return overview_list_growth

