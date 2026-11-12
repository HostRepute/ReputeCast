from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.connect_auth_result import ConnectAuthResult
  from ..models.connect_sender_option import ConnectSenderOption
  from ..models.connect_test_status_seed_list import ConnectTestStatusSeedList
  from ..models.connect_timeline_step import ConnectTimelineStep





T = TypeVar("T", bound="ConnectTestStatus")



@_attrs_define
class ConnectTestStatus:
    """ 
        Attributes:
            listening (bool):
            message_id (str):
            round_trip (str):
            seed_list (ConnectTestStatusSeedList):
            seeds (list[str]):
            sender_id (str):
            senders (list[ConnectSenderOption]):
            subject (str):
            timeline (list[ConnectTimelineStep]):
            to (str):
            auth (ConnectAuthResult | Unset):
     """

    listening: bool
    message_id: str
    round_trip: str
    seed_list: ConnectTestStatusSeedList
    seeds: list[str]
    sender_id: str
    senders: list[ConnectSenderOption]
    subject: str
    timeline: list[ConnectTimelineStep]
    to: str
    auth: ConnectAuthResult | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_auth_result import ConnectAuthResult # noqa: PLC0415
        from ..models.connect_sender_option import ConnectSenderOption # noqa: PLC0415
        from ..models.connect_test_status_seed_list import ConnectTestStatusSeedList # noqa: PLC0415
        from ..models.connect_timeline_step import ConnectTimelineStep # noqa: PLC0415
        listening = self.listening

        message_id = self.message_id

        round_trip = self.round_trip

        seed_list = self.seed_list.to_dict()

        seeds = self.seeds



        sender_id = self.sender_id

        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        subject = self.subject

        timeline = []
        for timeline_item_data in self.timeline:
            timeline_item = timeline_item_data.to_dict()
            timeline.append(timeline_item)



        to = self.to

        auth: dict[str, Any] | Unset = UNSET
        if not isinstance(self.auth, Unset):
            auth = self.auth.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "listening": listening,
            "messageId": message_id,
            "roundTrip": round_trip,
            "seedList": seed_list,
            "seeds": seeds,
            "senderId": sender_id,
            "senders": senders,
            "subject": subject,
            "timeline": timeline,
            "to": to,
        })
        if auth is not UNSET:
            field_dict["auth"] = auth

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_auth_result import ConnectAuthResult # noqa: PLC0415
        from ..models.connect_sender_option import ConnectSenderOption # noqa: PLC0415
        from ..models.connect_test_status_seed_list import ConnectTestStatusSeedList # noqa: PLC0415
        from ..models.connect_timeline_step import ConnectTimelineStep # noqa: PLC0415
        d = dict(src_dict)
        listening = d.pop("listening")

        message_id = d.pop("messageId")

        round_trip = d.pop("roundTrip")

        seed_list = ConnectTestStatusSeedList.from_dict(d.pop("seedList"))




        seeds = cast(list[str], d.pop("seeds"))


        sender_id = d.pop("senderId")

        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = ConnectSenderOption.from_dict(senders_item_data)



            senders.append(senders_item)


        subject = d.pop("subject")

        timeline = []
        _timeline = d.pop("timeline")
        for timeline_item_data in (_timeline):
            timeline_item = ConnectTimelineStep.from_dict(timeline_item_data)



            timeline.append(timeline_item)


        to = d.pop("to")

        _auth = d.pop("auth", UNSET)
        auth: ConnectAuthResult | Unset
        if isinstance(_auth,  Unset):
            auth = UNSET
        else:
            auth = ConnectAuthResult.from_dict(_auth)




        connect_test_status = cls(
            listening=listening,
            message_id=message_id,
            round_trip=round_trip,
            seed_list=seed_list,
            seeds=seeds,
            sender_id=sender_id,
            senders=senders,
            subject=subject,
            timeline=timeline,
            to=to,
            auth=auth,
        )

        return connect_test_status

