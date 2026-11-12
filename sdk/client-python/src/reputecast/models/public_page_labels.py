from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.public_page_label import PublicPageLabel





T = TypeVar("T", bound="PublicPageLabels")



@_attrs_define
class PublicPageLabels:
    """ 
        Attributes:
            by (str): Who wrote them last
            labels (list[PublicPageLabel]):
            language (str):
            shipped (bool):
            total (int):
            written (int): Labels with text in the language (all of them when they ship)
            at (datetime.datetime | Unset):
     """

    by: str
    labels: list[PublicPageLabel]
    language: str
    shipped: bool
    total: int
    written: int
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_page_label import PublicPageLabel # noqa: PLC0415
        by = self.by

        labels = []
        for labels_item_data in self.labels:
            labels_item = labels_item_data.to_dict()
            labels.append(labels_item)



        language = self.language

        shipped = self.shipped

        total = self.total

        written = self.written

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "by": by,
            "labels": labels,
            "language": language,
            "shipped": shipped,
            "total": total,
            "written": written,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_page_label import PublicPageLabel # noqa: PLC0415
        d = dict(src_dict)
        by = d.pop("by")

        labels = []
        _labels = d.pop("labels")
        for labels_item_data in (_labels):
            labels_item = PublicPageLabel.from_dict(labels_item_data)



            labels.append(labels_item)


        language = d.pop("language")

        shipped = d.pop("shipped")

        total = d.pop("total")

        written = d.pop("written")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        public_page_labels = cls(
            by=by,
            labels=labels,
            language=language,
            shipped=shipped,
            total=total,
            written=written,
            at=at,
        )

        return public_page_labels

