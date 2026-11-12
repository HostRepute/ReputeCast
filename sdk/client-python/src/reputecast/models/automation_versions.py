from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_version_item import AutomationVersionItem





T = TypeVar("T", bound="AutomationVersions")



@_attrs_define
class AutomationVersions:
    """ 
        Attributes:
            name (str):
            versions (list[AutomationVersionItem]):
     """

    name: str
    versions: list[AutomationVersionItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_version_item import AutomationVersionItem # noqa: PLC0415
        name = self.name

        versions = []
        for versions_item_data in self.versions:
            versions_item = versions_item_data.to_dict()
            versions.append(versions_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "versions": versions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_version_item import AutomationVersionItem # noqa: PLC0415
        d = dict(src_dict)
        name = d.pop("name")

        versions = []
        _versions = d.pop("versions")
        for versions_item_data in (_versions):
            versions_item = AutomationVersionItem.from_dict(versions_item_data)



            versions.append(versions_item)


        automation_versions = cls(
            name=name,
            versions=versions,
        )

        return automation_versions

