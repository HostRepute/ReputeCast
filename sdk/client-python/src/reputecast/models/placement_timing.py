from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_timing_folder import PlacementTimingFolder






T = TypeVar("T", bound="PlacementTiming")



@_attrs_define
class PlacementTiming:
    """ 
        Attributes:
            folder (PlacementTimingFolder):
            folder_name (str):
            ms (int):
            seed_provider (str):
            seed_provider_name (str):
     """

    folder: PlacementTimingFolder
    folder_name: str
    ms: int
    seed_provider: str
    seed_provider_name: str





    def to_dict(self) -> dict[str, Any]:
        folder = self.folder.value

        folder_name = self.folder_name

        ms = self.ms

        seed_provider = self.seed_provider

        seed_provider_name = self.seed_provider_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "folder": folder,
            "folderName": folder_name,
            "ms": ms,
            "seedProvider": seed_provider,
            "seedProviderName": seed_provider_name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        folder = PlacementTimingFolder(d.pop("folder"))




        folder_name = d.pop("folderName")

        ms = d.pop("ms")

        seed_provider = d.pop("seedProvider")

        seed_provider_name = d.pop("seedProviderName")

        placement_timing = cls(
            folder=folder,
            folder_name=folder_name,
            ms=ms,
            seed_provider=seed_provider,
            seed_provider_name=seed_provider_name,
        )

        return placement_timing

