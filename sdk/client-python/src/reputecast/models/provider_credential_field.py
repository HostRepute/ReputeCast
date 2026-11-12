from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderCredentialField")



@_attrs_define
class ProviderCredentialField:
    """ 
        Attributes:
            key (str):
            label (str):
            set_ (bool):
     """

    key: str
    label: str
    set_: bool





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        set_ = self.set_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "set": set_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        set_ = d.pop("set")

        provider_credential_field = cls(
            key=key,
            label=label,
            set_=set_,
        )

        return provider_credential_field

