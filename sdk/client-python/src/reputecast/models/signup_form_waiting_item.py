from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="SignupFormWaitingItem")



@_attrs_define
class SignupFormWaitingItem:
    """ 
        Attributes:
            attempts (int): How many times sending the email was tried
            created_at (datetime.datetime):
            email (str):
            first_name (str):
            id (str): The submission
            lists (list[str]): The lists the signup waits to join
     """

    attempts: int
    created_at: datetime.datetime
    email: str
    first_name: str
    id: str
    lists: list[str]





    def to_dict(self) -> dict[str, Any]:
        attempts = self.attempts

        created_at = self.created_at.isoformat()

        email = self.email

        first_name = self.first_name

        id = self.id

        lists = self.lists




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attempts": attempts,
            "createdAt": created_at,
            "email": email,
            "firstName": first_name,
            "id": id,
            "lists": lists,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attempts = d.pop("attempts")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        email = d.pop("email")

        first_name = d.pop("firstName")

        id = d.pop("id")

        lists = cast(list[str], d.pop("lists"))


        signup_form_waiting_item = cls(
            attempts=attempts,
            created_at=created_at,
            email=email,
            first_name=first_name,
            id=id,
            lists=lists,
        )

        return signup_form_waiting_item

