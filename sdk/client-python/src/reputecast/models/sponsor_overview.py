from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_overview_range import SponsorOverviewRange
from typing import cast

if TYPE_CHECKING:
  from ..models.sponsor_issue_row import SponsorIssueRow
  from ..models.sponsor_item import SponsorItem
  from ..models.sponsor_notice import SponsorNotice
  from ..models.sponsor_slot_item import SponsorSlotItem
  from ..models.sponsor_tiles import SponsorTiles





T = TypeVar("T", bound="SponsorOverview")



@_attrs_define
class SponsorOverview:
    """ 
        Attributes:
            can_manage (bool):
            currency (str):
            helpers (list[str]):
            library (list[SponsorItem]):
            notices (list[SponsorNotice]):
            past (list[SponsorIssueRow]):
            range_ (SponsorOverviewRange):
            slot_id (str):
            slots (list[SponsorSlotItem]):
            tiles (SponsorTiles):
            upcoming (list[SponsorIssueRow]):
     """

    can_manage: bool
    currency: str
    helpers: list[str]
    library: list[SponsorItem]
    notices: list[SponsorNotice]
    past: list[SponsorIssueRow]
    range_: SponsorOverviewRange
    slot_id: str
    slots: list[SponsorSlotItem]
    tiles: SponsorTiles
    upcoming: list[SponsorIssueRow]





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_issue_row import SponsorIssueRow # noqa: PLC0415
        from ..models.sponsor_item import SponsorItem # noqa: PLC0415
        from ..models.sponsor_notice import SponsorNotice # noqa: PLC0415
        from ..models.sponsor_slot_item import SponsorSlotItem # noqa: PLC0415
        from ..models.sponsor_tiles import SponsorTiles # noqa: PLC0415
        can_manage = self.can_manage

        currency = self.currency

        helpers = self.helpers



        library = []
        for library_item_data in self.library:
            library_item = library_item_data.to_dict()
            library.append(library_item)



        notices = []
        for notices_item_data in self.notices:
            notices_item = notices_item_data.to_dict()
            notices.append(notices_item)



        past = []
        for past_item_data in self.past:
            past_item = past_item_data.to_dict()
            past.append(past_item)



        range_ = self.range_.value

        slot_id = self.slot_id

        slots = []
        for slots_item_data in self.slots:
            slots_item = slots_item_data.to_dict()
            slots.append(slots_item)



        tiles = self.tiles.to_dict()

        upcoming = []
        for upcoming_item_data in self.upcoming:
            upcoming_item = upcoming_item_data.to_dict()
            upcoming.append(upcoming_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "currency": currency,
            "helpers": helpers,
            "library": library,
            "notices": notices,
            "past": past,
            "range": range_,
            "slotId": slot_id,
            "slots": slots,
            "tiles": tiles,
            "upcoming": upcoming,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_issue_row import SponsorIssueRow # noqa: PLC0415
        from ..models.sponsor_item import SponsorItem # noqa: PLC0415
        from ..models.sponsor_notice import SponsorNotice # noqa: PLC0415
        from ..models.sponsor_slot_item import SponsorSlotItem # noqa: PLC0415
        from ..models.sponsor_tiles import SponsorTiles # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        currency = d.pop("currency")

        helpers = cast(list[str], d.pop("helpers"))


        library = []
        _library = d.pop("library")
        for library_item_data in (_library):
            library_item = SponsorItem.from_dict(library_item_data)



            library.append(library_item)


        notices = []
        _notices = d.pop("notices")
        for notices_item_data in (_notices):
            notices_item = SponsorNotice.from_dict(notices_item_data)



            notices.append(notices_item)


        past = []
        _past = d.pop("past")
        for past_item_data in (_past):
            past_item = SponsorIssueRow.from_dict(past_item_data)



            past.append(past_item)


        range_ = SponsorOverviewRange(d.pop("range"))




        slot_id = d.pop("slotId")

        slots = []
        _slots = d.pop("slots")
        for slots_item_data in (_slots):
            slots_item = SponsorSlotItem.from_dict(slots_item_data)



            slots.append(slots_item)


        tiles = SponsorTiles.from_dict(d.pop("tiles"))




        upcoming = []
        _upcoming = d.pop("upcoming")
        for upcoming_item_data in (_upcoming):
            upcoming_item = SponsorIssueRow.from_dict(upcoming_item_data)



            upcoming.append(upcoming_item)


        sponsor_overview = cls(
            can_manage=can_manage,
            currency=currency,
            helpers=helpers,
            library=library,
            notices=notices,
            past=past,
            range_=range_,
            slot_id=slot_id,
            slots=slots,
            tiles=tiles,
            upcoming=upcoming,
        )

        return sponsor_overview

