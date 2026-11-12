from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_detail_mode import SegmentDetailMode
from ..models.segment_detail_refresh import SegmentDetailRefresh
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.segment_definition import SegmentDefinition
  from ..models.segment_tier_mix import SegmentTierMix





T = TypeVar("T", bound="SegmentDetail")



@_attrs_define
class SegmentDetail:
    """ 
        Attributes:
            description (str):
            generated_sql (str):
            id (str):
            mode (SegmentDetailMode):
            name (str):
            refresh (SegmentDetailRefresh):
            sql (str):
            tiers (SegmentTierMix):
            updated_at (datetime.datetime):
            updated_by (str):
            count (int | Unset):
            count_ms (int | Unset):
            counted_at (datetime.datetime | Unset):
            definition (SegmentDefinition | Unset):
     """

    description: str
    generated_sql: str
    id: str
    mode: SegmentDetailMode
    name: str
    refresh: SegmentDetailRefresh
    sql: str
    tiers: SegmentTierMix
    updated_at: datetime.datetime
    updated_by: str
    count: int | Unset = UNSET
    count_ms: int | Unset = UNSET
    counted_at: datetime.datetime | Unset = UNSET
    definition: SegmentDefinition | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_definition import SegmentDefinition # noqa: PLC0415
        from ..models.segment_tier_mix import SegmentTierMix # noqa: PLC0415
        description = self.description

        generated_sql = self.generated_sql

        id = self.id

        mode = self.mode.value

        name = self.name

        refresh = self.refresh.value

        sql = self.sql

        tiers = self.tiers.to_dict()

        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        count = self.count

        count_ms = self.count_ms

        counted_at: str | Unset = UNSET
        if not isinstance(self.counted_at, Unset):
            counted_at = self.counted_at.isoformat()

        definition: dict[str, Any] | Unset = UNSET
        if not isinstance(self.definition, Unset):
            definition = self.definition.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "generatedSql": generated_sql,
            "id": id,
            "mode": mode,
            "name": name,
            "refresh": refresh,
            "sql": sql,
            "tiers": tiers,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
        })
        if count is not UNSET:
            field_dict["count"] = count
        if count_ms is not UNSET:
            field_dict["countMs"] = count_ms
        if counted_at is not UNSET:
            field_dict["countedAt"] = counted_at
        if definition is not UNSET:
            field_dict["definition"] = definition

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_definition import SegmentDefinition # noqa: PLC0415
        from ..models.segment_tier_mix import SegmentTierMix # noqa: PLC0415
        d = dict(src_dict)
        description = d.pop("description")

        generated_sql = d.pop("generatedSql")

        id = d.pop("id")

        mode = SegmentDetailMode(d.pop("mode"))




        name = d.pop("name")

        refresh = SegmentDetailRefresh(d.pop("refresh"))




        sql = d.pop("sql")

        tiers = SegmentTierMix.from_dict(d.pop("tiers"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        count = d.pop("count", UNSET)

        count_ms = d.pop("countMs", UNSET)

        _counted_at = d.pop("countedAt", UNSET)
        counted_at: datetime.datetime | Unset
        if isinstance(_counted_at,  Unset):
            counted_at = UNSET
        else:
            counted_at = datetime.datetime.fromisoformat(_counted_at)




        _definition = d.pop("definition", UNSET)
        definition: SegmentDefinition | Unset
        if isinstance(_definition,  Unset):
            definition = UNSET
        else:
            definition = SegmentDefinition.from_dict(_definition)




        segment_detail = cls(
            description=description,
            generated_sql=generated_sql,
            id=id,
            mode=mode,
            name=name,
            refresh=refresh,
            sql=sql,
            tiers=tiers,
            updated_at=updated_at,
            updated_by=updated_by,
            count=count,
            count_ms=count_ms,
            counted_at=counted_at,
            definition=definition,
        )

        return segment_detail

