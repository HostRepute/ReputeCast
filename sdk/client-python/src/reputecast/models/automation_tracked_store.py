from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationTrackedStore")



@_attrs_define
class AutomationTrackedStore:
    """ 
        Attributes:
            host (str):
            store_id (str):
     """

    host: str
    store_id: str





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        store_id = self.store_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "storeId": store_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        store_id = d.pop("storeId")

        automation_tracked_store = cls(
            host=host,
            store_id=store_id,
        )

        return automation_tracked_store

