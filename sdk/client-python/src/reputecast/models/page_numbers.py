from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.page_result import PageResult





T = TypeVar("T", bound="PageNumbers")



@_attrs_define
class PageNumbers:
    """ 
        Attributes:
            result (PageResult):
            views (int):
     """

    result: PageResult
    views: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_result import PageResult # noqa: PLC0415
        result = self.result.to_dict()

        views = self.views


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "result": result,
            "views": views,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_result import PageResult # noqa: PLC0415
        d = dict(src_dict)
        result = PageResult.from_dict(d.pop("result"))




        views = d.pop("views")

        page_numbers = cls(
            result=result,
            views=views,
        )

        return page_numbers

