from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.tracker_store_option_platform import TrackerStoreOptionPlatform
from uuid import UUID






T = TypeVar("T", bound="TrackerStoreOption")



@_attrs_define
class TrackerStoreOption:
    """ 
        Attributes:
            id (UUID):
            name (str):
            platform (TrackerStoreOptionPlatform):
     """

    id: UUID
    name: str
    platform: TrackerStoreOptionPlatform





    def to_dict(self) -> dict[str, Any]:
        id = str(self.id)

        name = self.name

        platform = self.platform.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "platform": platform,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = UUID(d.pop("id"))




        name = d.pop("name")

        platform = TrackerStoreOptionPlatform(d.pop("platform"))




        tracker_store_option = cls(
            id=id,
            name=name,
            platform=platform,
        )

        return tracker_store_option

