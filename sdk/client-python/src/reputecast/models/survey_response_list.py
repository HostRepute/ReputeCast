from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.survey_response import SurveyResponse





T = TypeVar("T", bound="SurveyResponseList")



@_attrs_define
class SurveyResponseList:
    """ 
        Attributes:
            items (list[SurveyResponse]):
            names (bool):
            total (int):
     """

    items: list[SurveyResponse]
    names: bool
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_response import SurveyResponse # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        names = self.names

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "names": names,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_response import SurveyResponse # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SurveyResponse.from_dict(items_item_data)



            items.append(items_item)


        names = d.pop("names")

        total = d.pop("total")

        survey_response_list = cls(
            items=items,
            names=names,
            total=total,
        )

        return survey_response_list

