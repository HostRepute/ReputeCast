from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.grow_overview_program_status import GrowOverviewProgramStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.grow_need import GrowNeed
  from ..models.grow_overview_tiles import GrowOverviewTiles
  from ..models.grow_week import GrowWeek





T = TypeVar("T", bound="GrowOverview")



@_attrs_define
class GrowOverview:
    """ 
        Attributes:
            from_ (datetime.datetime): The period's first day (workspace time zone)
            helpers (GrowHelperSets):
            list_id (str):
            list_name (str):
            needs_you (list[GrowNeed]):
            program_id (str):
            program_status (GrowOverviewProgramStatus):
            recommended_by (int):
            recommends (int):
            tiles (GrowOverviewTiles):
            to (datetime.datetime): Today
            updated_at (datetime.datetime):
            weeks (list[GrowWeek]):
            program_start (datetime.datetime | Unset):
     """

    from_: datetime.datetime
    helpers: GrowHelperSets
    list_id: str
    list_name: str
    needs_you: list[GrowNeed]
    program_id: str
    program_status: GrowOverviewProgramStatus
    recommended_by: int
    recommends: int
    tiles: GrowOverviewTiles
    to: datetime.datetime
    updated_at: datetime.datetime
    weeks: list[GrowWeek]
    program_start: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.grow_need import GrowNeed # noqa: PLC0415
        from ..models.grow_overview_tiles import GrowOverviewTiles # noqa: PLC0415
        from ..models.grow_week import GrowWeek # noqa: PLC0415
        from_ = self.from_.isoformat()

        helpers = self.helpers.to_dict()

        list_id = self.list_id

        list_name = self.list_name

        needs_you = []
        for needs_you_item_data in self.needs_you:
            needs_you_item = needs_you_item_data.to_dict()
            needs_you.append(needs_you_item)



        program_id = self.program_id

        program_status = self.program_status.value

        recommended_by = self.recommended_by

        recommends = self.recommends

        tiles = self.tiles.to_dict()

        to = self.to.isoformat()

        updated_at = self.updated_at.isoformat()

        weeks = []
        for weeks_item_data in self.weeks:
            weeks_item = weeks_item_data.to_dict()
            weeks.append(weeks_item)



        program_start: str | Unset = UNSET
        if not isinstance(self.program_start, Unset):
            program_start = self.program_start.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "helpers": helpers,
            "listId": list_id,
            "listName": list_name,
            "needsYou": needs_you,
            "programId": program_id,
            "programStatus": program_status,
            "recommendedBy": recommended_by,
            "recommends": recommends,
            "tiles": tiles,
            "to": to,
            "updatedAt": updated_at,
            "weeks": weeks,
        })
        if program_start is not UNSET:
            field_dict["programStart"] = program_start

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.grow_need import GrowNeed # noqa: PLC0415
        from ..models.grow_overview_tiles import GrowOverviewTiles # noqa: PLC0415
        from ..models.grow_week import GrowWeek # noqa: PLC0415
        d = dict(src_dict)
        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        list_id = d.pop("listId")

        list_name = d.pop("listName")

        needs_you = []
        _needs_you = d.pop("needsYou")
        for needs_you_item_data in (_needs_you):
            needs_you_item = GrowNeed.from_dict(needs_you_item_data)



            needs_you.append(needs_you_item)


        program_id = d.pop("programId")

        program_status = GrowOverviewProgramStatus(d.pop("programStatus"))




        recommended_by = d.pop("recommendedBy")

        recommends = d.pop("recommends")

        tiles = GrowOverviewTiles.from_dict(d.pop("tiles"))




        to = datetime.datetime.fromisoformat(d.pop("to"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        weeks = []
        _weeks = d.pop("weeks")
        for weeks_item_data in (_weeks):
            weeks_item = GrowWeek.from_dict(weeks_item_data)



            weeks.append(weeks_item)


        _program_start = d.pop("programStart", UNSET)
        program_start: datetime.datetime | Unset
        if isinstance(_program_start,  Unset):
            program_start = UNSET
        else:
            program_start = datetime.datetime.fromisoformat(_program_start)




        grow_overview = cls(
            from_=from_,
            helpers=helpers,
            list_id=list_id,
            list_name=list_name,
            needs_you=needs_you,
            program_id=program_id,
            program_status=program_status,
            recommended_by=recommended_by,
            recommends=recommends,
            tiles=tiles,
            to=to,
            updated_at=updated_at,
            weeks=weeks,
            program_start=program_start,
        )

        return grow_overview

