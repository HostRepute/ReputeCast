from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="NewLinkInput")



@_attrs_define
class NewLinkInput:
    """ 
        Attributes:
            requested_by (str | Unset):
     """

    requested_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        requested_by = self.requested_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if requested_by is not UNSET:
            field_dict["requestedBy"] = requested_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        requested_by = d.pop("requestedBy", UNSET)

        new_link_input = cls(
            requested_by=requested_by,
        )

        return new_link_input

