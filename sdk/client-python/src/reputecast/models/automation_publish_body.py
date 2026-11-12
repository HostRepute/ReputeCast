from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_publish_body_migration import AutomationPublishBodyMigration






T = TypeVar("T", bound="AutomationPublishBody")



@_attrs_define
class AutomationPublishBody:
    """ 
        Attributes:
            migration (AutomationPublishBodyMigration): finish: contacts in the flow finish on the old version; move: they
                move to the new one at their current step
            version (int): The draft version reviewed in the publish dialog
     """

    migration: AutomationPublishBodyMigration
    version: int





    def to_dict(self) -> dict[str, Any]:
        migration = self.migration.value

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "migration": migration,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        migration = AutomationPublishBodyMigration(d.pop("migration"))




        version = d.pop("version")

        automation_publish_body = cls(
            migration=migration,
            version=version,
        )

        return automation_publish_body

