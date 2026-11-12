from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_program_detail_status import ReferralProgramDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.referral_counting import ReferralCounting
  from ..models.referral_email_settings import ReferralEmailSettings
  from ..models.referral_leaderboard_settings import ReferralLeaderboardSettings
  from ..models.referral_page_settings import ReferralPageSettings
  from ..models.referral_sender_option import ReferralSenderOption





T = TypeVar("T", bound="ReferralProgramDetail")



@_attrs_define
class ReferralProgramDetail:
    """ 
        Attributes:
            announced_in (str):
            awaiting (int):
            counting (ReferralCounting):
            emails (ReferralEmailSettings):
            held (int):
            helpers (GrowHelperSets):
            id (str):
            in_hold (int):
            leaderboard (ReferralLeaderboardSettings):
            link_example (str):
            link_host (str):
            links (int):
            list_id (str):
            list_name (str):
            pages (ReferralPageSettings):
            senders (list[ReferralSenderOption]):
            setup_saved_by (str):
            setup_step (int):
            started_by (str):
            status (ReferralProgramDetailStatus):
            status_by (str):
            subscribers (int):
            updated_at (datetime.datetime):
            held_oldest (datetime.datetime | Unset):
            setup_saved_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            status_at (datetime.datetime | Unset):
     """

    announced_in: str
    awaiting: int
    counting: ReferralCounting
    emails: ReferralEmailSettings
    held: int
    helpers: GrowHelperSets
    id: str
    in_hold: int
    leaderboard: ReferralLeaderboardSettings
    link_example: str
    link_host: str
    links: int
    list_id: str
    list_name: str
    pages: ReferralPageSettings
    senders: list[ReferralSenderOption]
    setup_saved_by: str
    setup_step: int
    started_by: str
    status: ReferralProgramDetailStatus
    status_by: str
    subscribers: int
    updated_at: datetime.datetime
    held_oldest: datetime.datetime | Unset = UNSET
    setup_saved_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    status_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_counting import ReferralCounting # noqa: PLC0415
        from ..models.referral_email_settings import ReferralEmailSettings # noqa: PLC0415
        from ..models.referral_leaderboard_settings import ReferralLeaderboardSettings # noqa: PLC0415
        from ..models.referral_page_settings import ReferralPageSettings # noqa: PLC0415
        from ..models.referral_sender_option import ReferralSenderOption # noqa: PLC0415
        announced_in = self.announced_in

        awaiting = self.awaiting

        counting = self.counting.to_dict()

        emails = self.emails.to_dict()

        held = self.held

        helpers = self.helpers.to_dict()

        id = self.id

        in_hold = self.in_hold

        leaderboard = self.leaderboard.to_dict()

        link_example = self.link_example

        link_host = self.link_host

        links = self.links

        list_id = self.list_id

        list_name = self.list_name

        pages = self.pages.to_dict()

        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        setup_saved_by = self.setup_saved_by

        setup_step = self.setup_step

        started_by = self.started_by

        status = self.status.value

        status_by = self.status_by

        subscribers = self.subscribers

        updated_at = self.updated_at.isoformat()

        held_oldest: str | Unset = UNSET
        if not isinstance(self.held_oldest, Unset):
            held_oldest = self.held_oldest.isoformat()

        setup_saved_at: str | Unset = UNSET
        if not isinstance(self.setup_saved_at, Unset):
            setup_saved_at = self.setup_saved_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        status_at: str | Unset = UNSET
        if not isinstance(self.status_at, Unset):
            status_at = self.status_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "announcedIn": announced_in,
            "awaiting": awaiting,
            "counting": counting,
            "emails": emails,
            "held": held,
            "helpers": helpers,
            "id": id,
            "inHold": in_hold,
            "leaderboard": leaderboard,
            "linkExample": link_example,
            "linkHost": link_host,
            "links": links,
            "listId": list_id,
            "listName": list_name,
            "pages": pages,
            "senders": senders,
            "setupSavedBy": setup_saved_by,
            "setupStep": setup_step,
            "startedBy": started_by,
            "status": status,
            "statusBy": status_by,
            "subscribers": subscribers,
            "updatedAt": updated_at,
        })
        if held_oldest is not UNSET:
            field_dict["heldOldest"] = held_oldest
        if setup_saved_at is not UNSET:
            field_dict["setupSavedAt"] = setup_saved_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if status_at is not UNSET:
            field_dict["statusAt"] = status_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_counting import ReferralCounting # noqa: PLC0415
        from ..models.referral_email_settings import ReferralEmailSettings # noqa: PLC0415
        from ..models.referral_leaderboard_settings import ReferralLeaderboardSettings # noqa: PLC0415
        from ..models.referral_page_settings import ReferralPageSettings # noqa: PLC0415
        from ..models.referral_sender_option import ReferralSenderOption # noqa: PLC0415
        d = dict(src_dict)
        announced_in = d.pop("announcedIn")

        awaiting = d.pop("awaiting")

        counting = ReferralCounting.from_dict(d.pop("counting"))




        emails = ReferralEmailSettings.from_dict(d.pop("emails"))




        held = d.pop("held")

        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        id = d.pop("id")

        in_hold = d.pop("inHold")

        leaderboard = ReferralLeaderboardSettings.from_dict(d.pop("leaderboard"))




        link_example = d.pop("linkExample")

        link_host = d.pop("linkHost")

        links = d.pop("links")

        list_id = d.pop("listId")

        list_name = d.pop("listName")

        pages = ReferralPageSettings.from_dict(d.pop("pages"))




        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = ReferralSenderOption.from_dict(senders_item_data)



            senders.append(senders_item)


        setup_saved_by = d.pop("setupSavedBy")

        setup_step = d.pop("setupStep")

        started_by = d.pop("startedBy")

        status = ReferralProgramDetailStatus(d.pop("status"))




        status_by = d.pop("statusBy")

        subscribers = d.pop("subscribers")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        _held_oldest = d.pop("heldOldest", UNSET)
        held_oldest: datetime.datetime | Unset
        if isinstance(_held_oldest,  Unset):
            held_oldest = UNSET
        else:
            held_oldest = datetime.datetime.fromisoformat(_held_oldest)




        _setup_saved_at = d.pop("setupSavedAt", UNSET)
        setup_saved_at: datetime.datetime | Unset
        if isinstance(_setup_saved_at,  Unset):
            setup_saved_at = UNSET
        else:
            setup_saved_at = datetime.datetime.fromisoformat(_setup_saved_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        _status_at = d.pop("statusAt", UNSET)
        status_at: datetime.datetime | Unset
        if isinstance(_status_at,  Unset):
            status_at = UNSET
        else:
            status_at = datetime.datetime.fromisoformat(_status_at)




        referral_program_detail = cls(
            announced_in=announced_in,
            awaiting=awaiting,
            counting=counting,
            emails=emails,
            held=held,
            helpers=helpers,
            id=id,
            in_hold=in_hold,
            leaderboard=leaderboard,
            link_example=link_example,
            link_host=link_host,
            links=links,
            list_id=list_id,
            list_name=list_name,
            pages=pages,
            senders=senders,
            setup_saved_by=setup_saved_by,
            setup_step=setup_step,
            started_by=started_by,
            status=status,
            status_by=status_by,
            subscribers=subscribers,
            updated_at=updated_at,
            held_oldest=held_oldest,
            setup_saved_at=setup_saved_at,
            started_at=started_at,
            status_at=status_at,
        )

        return referral_program_detail

