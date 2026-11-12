from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.report_provider_column import ReportProviderColumn





T = TypeVar("T", bound="ReportProviderTable")



@_attrs_define
class ReportProviderTable:
    """ 
        Attributes:
            columns (list[ReportProviderColumn]):
            split (bool):
     """

    columns: list[ReportProviderColumn]
    split: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_provider_column import ReportProviderColumn # noqa: PLC0415
        columns = []
        for columns_item_data in self.columns:
            columns_item = columns_item_data.to_dict()
            columns.append(columns_item)



        split = self.split


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "columns": columns,
            "split": split,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_provider_column import ReportProviderColumn # noqa: PLC0415
        d = dict(src_dict)
        columns = []
        _columns = d.pop("columns")
        for columns_item_data in (_columns):
            columns_item = ReportProviderColumn.from_dict(columns_item_data)



            columns.append(columns_item)


        split = d.pop("split")

        report_provider_table = cls(
            columns=columns,
            split=split,
        )

        return report_provider_table

