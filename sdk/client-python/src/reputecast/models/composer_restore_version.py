from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ComposerRestoreVersion")



@_attrs_define
class ComposerRestoreVersion:
    """ 
        Attributes:
            base_version (int):
     """

    base_version: int





    def to_dict(self) -> dict[str, Any]:
        base_version = self.base_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseVersion": base_version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base_version = d.pop("baseVersion")

        composer_restore_version = cls(
            base_version=base_version,
        )

        return composer_restore_version

