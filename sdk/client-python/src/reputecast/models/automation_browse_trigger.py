from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="AutomationBrowseTrigger")



@_attrs_define
class AutomationBrowseTrigger:
    """ 
        Attributes:
            after_hours (int):
            min_views (int):
            store_id (str): The connected store (uuid) whose products the tracked site's events name
            collection_ids (list[str] | Unset):
     """

    after_hours: int
    min_views: int
    store_id: str
    collection_ids: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        after_hours = self.after_hours

        min_views = self.min_views

        store_id = self.store_id

        collection_ids: list[str] | Unset = UNSET
        if not isinstance(self.collection_ids, Unset):
            collection_ids = self.collection_ids




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "afterHours": after_hours,
            "minViews": min_views,
            "storeId": store_id,
        })
        if collection_ids is not UNSET:
            field_dict["collectionIds"] = collection_ids

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after_hours = d.pop("afterHours")

        min_views = d.pop("minViews")

        store_id = d.pop("storeId")

        collection_ids = cast(list[str], d.pop("collectionIds", UNSET))


        automation_browse_trigger = cls(
            after_hours=after_hours,
            min_views=min_views,
            store_id=store_id,
            collection_ids=collection_ids,
        )

        return automation_browse_trigger

