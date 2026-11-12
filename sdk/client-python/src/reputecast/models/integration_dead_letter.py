from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationDeadLetter")



@_attrs_define
class IntegrationDeadLetter:
    """ 
        Attributes:
            event (str):
            first_try_at (datetime.datetime):
            id (str):
            kept_until (datetime.datetime):
            email (str | Unset):
            error (str | Unset):
            last_answer (int | Unset):
            last_attempt (datetime.datetime | Unset):
            status_text (str | Unset):
     """

    event: str
    first_try_at: datetime.datetime
    id: str
    kept_until: datetime.datetime
    email: str | Unset = UNSET
    error: str | Unset = UNSET
    last_answer: int | Unset = UNSET
    last_attempt: datetime.datetime | Unset = UNSET
    status_text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        event = self.event

        first_try_at = self.first_try_at.isoformat()

        id = self.id

        kept_until = self.kept_until.isoformat()

        email = self.email

        error = self.error

        last_answer = self.last_answer

        last_attempt: str | Unset = UNSET
        if not isinstance(self.last_attempt, Unset):
            last_attempt = self.last_attempt.isoformat()

        status_text = self.status_text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "event": event,
            "firstTryAt": first_try_at,
            "id": id,
            "keptUntil": kept_until,
        })
        if email is not UNSET:
            field_dict["email"] = email
        if error is not UNSET:
            field_dict["error"] = error
        if last_answer is not UNSET:
            field_dict["lastAnswer"] = last_answer
        if last_attempt is not UNSET:
            field_dict["lastAttempt"] = last_attempt
        if status_text is not UNSET:
            field_dict["statusText"] = status_text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        event = d.pop("event")

        first_try_at = datetime.datetime.fromisoformat(d.pop("firstTryAt"))




        id = d.pop("id")

        kept_until = datetime.datetime.fromisoformat(d.pop("keptUntil"))




        email = d.pop("email", UNSET)

        error = d.pop("error", UNSET)

        last_answer = d.pop("lastAnswer", UNSET)

        _last_attempt = d.pop("lastAttempt", UNSET)
        last_attempt: datetime.datetime | Unset
        if isinstance(_last_attempt,  Unset):
            last_attempt = UNSET
        else:
            last_attempt = datetime.datetime.fromisoformat(_last_attempt)




        status_text = d.pop("statusText", UNSET)

        integration_dead_letter = cls(
            event=event,
            first_try_at=first_try_at,
            id=id,
            kept_until=kept_until,
            email=email,
            error=error,
            last_answer=last_answer,
            last_attempt=last_attempt,
            status_text=status_text,
        )

        return integration_dead_letter

