from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ListRestore")



@_attrs_define
class ListRestore:
    """ 
        Attributes:
            automations (list[str]):
            campaigns (list[str]):
            forms (list[str]):
            id (str):
            left (list[str]):
            name (str):
     """

    automations: list[str]
    campaigns: list[str]
    forms: list[str]
    id: str
    left: list[str]
    name: str





    def to_dict(self) -> dict[str, Any]:
        automations = self.automations



        campaigns = self.campaigns



        forms = self.forms



        id = self.id

        left = self.left



        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automations": automations,
            "campaigns": campaigns,
            "forms": forms,
            "id": id,
            "left": left,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automations = cast(list[str], d.pop("automations"))


        campaigns = cast(list[str], d.pop("campaigns"))


        forms = cast(list[str], d.pop("forms"))


        id = d.pop("id")

        left = cast(list[str], d.pop("left"))


        name = d.pop("name")

        list_restore = cls(
            automations=automations,
            campaigns=campaigns,
            forms=forms,
            id=id,
            left=left,
            name=name,
        )

        return list_restore

