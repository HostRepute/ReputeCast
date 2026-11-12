from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="InboundSetupField")



@_attrs_define
class InboundSetupField:
    """ 
        Attributes:
            key (str):
            label (str):
            required (bool):
            value (str):
            help_ (str | Unset):
     """

    key: str
    label: str
    required: bool
    value: str
    help_: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        required = self.required

        value = self.value

        help_ = self.help_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "required": required,
            "value": value,
        })
        if help_ is not UNSET:
            field_dict["help"] = help_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        required = d.pop("required")

        value = d.pop("value")

        help_ = d.pop("help", UNSET)

        inbound_setup_field = cls(
            key=key,
            label=label,
            required=required,
            value=value,
            help_=help_,
        )

        return inbound_setup_field

