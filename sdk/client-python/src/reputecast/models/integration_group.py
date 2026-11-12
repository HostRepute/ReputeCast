from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_group_key import IntegrationGroupKey
from typing import cast

if TYPE_CHECKING:
  from ..models.integration_entry import IntegrationEntry





T = TypeVar("T", bound="IntegrationGroup")



@_attrs_define
class IntegrationGroup:
    """ 
        Attributes:
            entries (list[IntegrationEntry]):
            key (IntegrationGroupKey):
     """

    entries: list[IntegrationEntry]
    key: IntegrationGroupKey





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_entry import IntegrationEntry # noqa: PLC0415
        entries = []
        for entries_item_data in self.entries:
            entries_item = entries_item_data.to_dict()
            entries.append(entries_item)



        key = self.key.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "entries": entries,
            "key": key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_entry import IntegrationEntry # noqa: PLC0415
        d = dict(src_dict)
        entries = []
        _entries = d.pop("entries")
        for entries_item_data in (_entries):
            entries_item = IntegrationEntry.from_dict(entries_item_data)



            entries.append(entries_item)


        key = IntegrationGroupKey(d.pop("key"))




        integration_group = cls(
            entries=entries,
            key=key,
        )

        return integration_group

