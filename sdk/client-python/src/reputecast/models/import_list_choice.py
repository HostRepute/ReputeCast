from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ImportListChoice")



@_attrs_define
class ImportListChoice:
    """ 
        Attributes:
            source_id (str):
            target (str): List id, or new:<name>
     """

    source_id: str
    target: str





    def to_dict(self) -> dict[str, Any]:
        source_id = self.source_id

        target = self.target


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "sourceId": source_id,
            "target": target,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source_id = d.pop("sourceId")

        target = d.pop("target")

        import_list_choice = cls(
            source_id=source_id,
            target=target,
        )

        return import_list_choice

