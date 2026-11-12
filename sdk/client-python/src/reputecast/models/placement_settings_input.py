from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_settings_input_delete_after import PlacementSettingsInputDeleteAfter






T = TypeVar("T", bound="PlacementSettingsInput")



@_attrs_define
class PlacementSettingsInput:
    """ 
        Attributes:
            delete_after (PlacementSettingsInputDeleteAfter):
     """

    delete_after: PlacementSettingsInputDeleteAfter





    def to_dict(self) -> dict[str, Any]:
        delete_after = self.delete_after.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "deleteAfter": delete_after,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        delete_after = PlacementSettingsInputDeleteAfter(d.pop("deleteAfter"))




        placement_settings_input = cls(
            delete_after=delete_after,
        )

        return placement_settings_input

