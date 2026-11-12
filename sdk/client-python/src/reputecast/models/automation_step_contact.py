from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_step_contact_wait import AutomationStepContactWait
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AutomationStepContact")



@_attrs_define
class AutomationStepContact:
    """ 
        Attributes:
            contact_id (str):
            email (str):
            entered_at (datetime.datetime):
            name (str):
            run_id (str):
            since (datetime.datetime):
            wait (AutomationStepContactWait):
            until (datetime.datetime | Unset):
     """

    contact_id: str
    email: str
    entered_at: datetime.datetime
    name: str
    run_id: str
    since: datetime.datetime
    wait: AutomationStepContactWait
    until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact_id = self.contact_id

        email = self.email

        entered_at = self.entered_at.isoformat()

        name = self.name

        run_id = self.run_id

        since = self.since.isoformat()

        wait = self.wait.value

        until: str | Unset = UNSET
        if not isinstance(self.until, Unset):
            until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contactId": contact_id,
            "email": email,
            "enteredAt": entered_at,
            "name": name,
            "runId": run_id,
            "since": since,
            "wait": wait,
        })
        if until is not UNSET:
            field_dict["until"] = until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact_id = d.pop("contactId")

        email = d.pop("email")

        entered_at = datetime.datetime.fromisoformat(d.pop("enteredAt"))




        name = d.pop("name")

        run_id = d.pop("runId")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        wait = AutomationStepContactWait(d.pop("wait"))




        _until = d.pop("until", UNSET)
        until: datetime.datetime | Unset
        if isinstance(_until,  Unset):
            until = UNSET
        else:
            until = datetime.datetime.fromisoformat(_until)




        automation_step_contact = cls(
            contact_id=contact_id,
            email=email,
            entered_at=entered_at,
            name=name,
            run_id=run_id,
            since=since,
            wait=wait,
            until=until,
        )

        return automation_step_contact

