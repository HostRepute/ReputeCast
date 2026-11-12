from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AudienceListInput")



@_attrs_define
class AudienceListInput:
    """ 
        Attributes:
            name (str):
            description (str | Unset):
            double_opt_in (bool | Unset): New signups confirm by email first (default true)
     """

    name: str
    description: str | Unset = UNSET
    double_opt_in: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        description = self.description

        double_opt_in = self.double_opt_in


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if description is not UNSET:
            field_dict["description"] = description
        if double_opt_in is not UNSET:
            field_dict["doubleOptIn"] = double_opt_in

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        description = d.pop("description", UNSET)

        double_opt_in = d.pop("doubleOptIn", UNSET)

        audience_list_input = cls(
            name=name,
            description=description,
            double_opt_in=double_opt_in,
        )

        return audience_list_input

