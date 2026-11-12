from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_site_update_worker_mode import PushSiteUpdateWorkerMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="PushSiteUpdate")



@_attrs_define
class PushSiteUpdate:
    """ 
        Attributes:
            label (str):
            worker_mode (PushSiteUpdateWorkerMode):
            worker_path (str):
            icon_media_id (str | Unset):
     """

    label: str
    worker_mode: PushSiteUpdateWorkerMode
    worker_path: str
    icon_media_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        worker_mode = self.worker_mode.value

        worker_path = self.worker_path

        icon_media_id = self.icon_media_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "workerMode": worker_mode,
            "workerPath": worker_path,
        })
        if icon_media_id is not UNSET:
            field_dict["iconMediaId"] = icon_media_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        worker_mode = PushSiteUpdateWorkerMode(d.pop("workerMode"))




        worker_path = d.pop("workerPath")

        icon_media_id = d.pop("iconMediaId", UNSET)

        push_site_update = cls(
            label=label,
            worker_mode=worker_mode,
            worker_path=worker_path,
            icon_media_id=icon_media_id,
        )

        return push_site_update

