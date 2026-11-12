from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalLogTemplate")



@_attrs_define
class TransactionalLogTemplate:
    """ 
        Attributes:
            key (str):
            name (str):
            version (int | Unset):
     """

    key: str
    name: str
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        name = self.name

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "name": name,
        })
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        name = d.pop("name")

        version = d.pop("version", UNSET)

        transactional_log_template = cls(
            key=key,
            name=name,
            version=version,
        )

        return transactional_log_template

