from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_profile_field_display import ContactProfileFieldDisplay
from ..models.contact_profile_field_type import ContactProfileFieldType






T = TypeVar("T", bound="ContactProfileField")



@_attrs_define
class ContactProfileField:
    """ 
        Attributes:
            custom (bool):
            display (ContactProfileFieldDisplay):
            key (str):
            label (str):
            type_ (ContactProfileFieldType):
            value (str):
     """

    custom: bool
    display: ContactProfileFieldDisplay
    key: str
    label: str
    type_: ContactProfileFieldType
    value: str





    def to_dict(self) -> dict[str, Any]:
        custom = self.custom

        display = self.display.value

        key = self.key

        label = self.label

        type_ = self.type_.value

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "custom": custom,
            "display": display,
            "key": key,
            "label": label,
            "type": type_,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        custom = d.pop("custom")

        display = ContactProfileFieldDisplay(d.pop("display"))




        key = d.pop("key")

        label = d.pop("label")

        type_ = ContactProfileFieldType(d.pop("type"))




        value = d.pop("value")

        contact_profile_field = cls(
            custom=custom,
            display=display,
            key=key,
            label=label,
            type_=type_,
            value=value,
        )

        return contact_profile_field

