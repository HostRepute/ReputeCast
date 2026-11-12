from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ConnectSetupInput")



@_attrs_define
class ConnectSetupInput:
    """ 
        Attributes:
            adapter (str):
     """

    adapter: str





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        connect_setup_input = cls(
            adapter=adapter,
        )

        return connect_setup_input

