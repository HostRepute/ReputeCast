from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VisibilityOption")



@_attrs_define
class VisibilityOption:
    """ 
        Attributes:
            label (str):
            value (str):
            contacts (int | Unset):
     """

    label: str
    value: str
    contacts: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        value = self.value

        contacts = self.contacts


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "value": value,
        })
        if contacts is not UNSET:
            field_dict["contacts"] = contacts

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        value = d.pop("value")

        contacts = d.pop("contacts", UNSET)

        visibility_option = cls(
            label=label,
            value=value,
            contacts=contacts,
        )

        return visibility_option

