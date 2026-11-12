from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.poll_source_kind import PollSourceKind
from ..models.poll_source_poll_kind import PollSourcePollKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.poll_option import PollOption





T = TypeVar("T", bound="PollSource")



@_attrs_define
class PollSource:
    """ 
        Attributes:
            answers (list[PollOption]):
            kind (PollSourceKind):
            name (str):
            ref (str):
            status (str):
            campaign (str | Unset):
            closes_at (datetime.datetime | Unset):
            poll_kind (PollSourcePollKind | Unset):
     """

    answers: list[PollOption]
    kind: PollSourceKind
    name: str
    ref: str
    status: str
    campaign: str | Unset = UNSET
    closes_at: datetime.datetime | Unset = UNSET
    poll_kind: PollSourcePollKind | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.poll_option import PollOption # noqa: PLC0415
        answers = []
        for answers_item_data in self.answers:
            answers_item = answers_item_data.to_dict()
            answers.append(answers_item)



        kind = self.kind.value

        name = self.name

        ref = self.ref

        status = self.status

        campaign = self.campaign

        closes_at: str | Unset = UNSET
        if not isinstance(self.closes_at, Unset):
            closes_at = self.closes_at.isoformat()

        poll_kind: str | Unset = UNSET
        if not isinstance(self.poll_kind, Unset):
            poll_kind = self.poll_kind.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answers": answers,
            "kind": kind,
            "name": name,
            "ref": ref,
            "status": status,
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if closes_at is not UNSET:
            field_dict["closesAt"] = closes_at
        if poll_kind is not UNSET:
            field_dict["pollKind"] = poll_kind

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.poll_option import PollOption # noqa: PLC0415
        d = dict(src_dict)
        answers = []
        _answers = d.pop("answers")
        for answers_item_data in (_answers):
            answers_item = PollOption.from_dict(answers_item_data)



            answers.append(answers_item)


        kind = PollSourceKind(d.pop("kind"))




        name = d.pop("name")

        ref = d.pop("ref")

        status = d.pop("status")

        campaign = d.pop("campaign", UNSET)

        _closes_at = d.pop("closesAt", UNSET)
        closes_at: datetime.datetime | Unset
        if isinstance(_closes_at,  Unset):
            closes_at = UNSET
        else:
            closes_at = datetime.datetime.fromisoformat(_closes_at)




        _poll_kind = d.pop("pollKind", UNSET)
        poll_kind: PollSourcePollKind | Unset
        if isinstance(_poll_kind,  Unset):
            poll_kind = UNSET
        else:
            poll_kind = PollSourcePollKind(_poll_kind)




        poll_source = cls(
            answers=answers,
            kind=kind,
            name=name,
            ref=ref,
            status=status,
            campaign=campaign,
            closes_at=closes_at,
            poll_kind=poll_kind,
        )

        return poll_source

