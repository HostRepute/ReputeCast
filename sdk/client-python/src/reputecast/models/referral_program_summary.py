from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_program_summary_status import ReferralProgramSummaryStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralProgramSummary")



@_attrs_define
class ReferralProgramSummary:
    """ 
        Attributes:
            id (str):
            list_id (str):
            list_name (str):
            setup_step (int): The set-up guide's step (1 to 5) until the program is first turned on; 0 afterwards
            status (ReferralProgramSummaryStatus):
            subscribers (int):
            started_at (datetime.datetime | Unset):
     """

    id: str
    list_id: str
    list_name: str
    setup_step: int
    status: ReferralProgramSummaryStatus
    subscribers: int
    started_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        list_id = self.list_id

        list_name = self.list_name

        setup_step = self.setup_step

        status = self.status.value

        subscribers = self.subscribers

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "listId": list_id,
            "listName": list_name,
            "setupStep": setup_step,
            "status": status,
            "subscribers": subscribers,
        })
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        list_id = d.pop("listId")

        list_name = d.pop("listName")

        setup_step = d.pop("setupStep")

        status = ReferralProgramSummaryStatus(d.pop("status"))




        subscribers = d.pop("subscribers")

        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        referral_program_summary = cls(
            id=id,
            list_id=list_id,
            list_name=list_name,
            setup_step=setup_step,
            status=status,
            subscribers=subscribers,
            started_at=started_at,
        )

        return referral_program_summary

