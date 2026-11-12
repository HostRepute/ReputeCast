from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_audience_row import ReadinessAudienceRow





T = TypeVar("T", bound="ReadinessAudience")



@_attrs_define
class ReadinessAudience:
    """ 
        Attributes:
            duplicates (int):
            excluded_unengaged (int):
            final (int):
            lists (list[str]):
            role (int):
            rows (list[ReadinessAudienceRow]):
            suppressed (int):
            unengaged (int):
            unengaged_excluded (bool):
     """

    duplicates: int
    excluded_unengaged: int
    final: int
    lists: list[str]
    role: int
    rows: list[ReadinessAudienceRow]
    suppressed: int
    unengaged: int
    unengaged_excluded: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_audience_row import ReadinessAudienceRow # noqa: PLC0415
        duplicates = self.duplicates

        excluded_unengaged = self.excluded_unengaged

        final = self.final

        lists = self.lists



        role = self.role

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        suppressed = self.suppressed

        unengaged = self.unengaged

        unengaged_excluded = self.unengaged_excluded


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "duplicates": duplicates,
            "excludedUnengaged": excluded_unengaged,
            "final": final,
            "lists": lists,
            "role": role,
            "rows": rows,
            "suppressed": suppressed,
            "unengaged": unengaged,
            "unengagedExcluded": unengaged_excluded,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_audience_row import ReadinessAudienceRow # noqa: PLC0415
        d = dict(src_dict)
        duplicates = d.pop("duplicates")

        excluded_unengaged = d.pop("excludedUnengaged")

        final = d.pop("final")

        lists = cast(list[str], d.pop("lists"))


        role = d.pop("role")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = ReadinessAudienceRow.from_dict(rows_item_data)



            rows.append(rows_item)


        suppressed = d.pop("suppressed")

        unengaged = d.pop("unengaged")

        unengaged_excluded = d.pop("unengagedExcluded")

        readiness_audience = cls(
            duplicates=duplicates,
            excluded_unengaged=excluded_unengaged,
            final=final,
            lists=lists,
            role=role,
            rows=rows,
            suppressed=suppressed,
            unengaged=unengaged,
            unengaged_excluded=unengaged_excluded,
        )

        return readiness_audience

