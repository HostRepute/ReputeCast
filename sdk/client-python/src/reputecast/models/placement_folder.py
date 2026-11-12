from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_folder_role import PlacementFolderRole






T = TypeVar("T", bound="PlacementFolder")



@_attrs_define
class PlacementFolder:
    """ 
        Attributes:
            label (str):
            name (str):
            role (PlacementFolderRole):
     """

    label: str
    name: str
    role: PlacementFolderRole





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        name = self.name

        role = self.role.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "name": name,
            "role": role,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        name = d.pop("name")

        role = PlacementFolderRole(d.pop("role"))




        placement_folder = cls(
            label=label,
            name=name,
            role=role,
        )

        return placement_folder

