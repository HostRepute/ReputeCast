from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.public_page_labels_input_labels import PublicPageLabelsInputLabels





T = TypeVar("T", bound="PublicPageLabelsInput")



@_attrs_define
class PublicPageLabelsInput:
    """ 
        Attributes:
            labels (PublicPageLabelsInputLabels):
     """

    labels: PublicPageLabelsInputLabels





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_page_labels_input_labels import PublicPageLabelsInputLabels # noqa: PLC0415
        labels = self.labels.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "labels": labels,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_page_labels_input_labels import PublicPageLabelsInputLabels # noqa: PLC0415
        d = dict(src_dict)
        labels = PublicPageLabelsInputLabels.from_dict(d.pop("labels"))




        public_page_labels_input = cls(
            labels=labels,
        )

        return public_page_labels_input

