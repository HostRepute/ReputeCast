from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.ai_personal_account import AIPersonalAccount





T = TypeVar("T", bound="AIPersonalPerson")



@_attrs_define
class AIPersonalPerson:
    """ 
        Attributes:
            accounts (list[AIPersonalAccount]):
            cost_micros (int):
            name (str):
            requests (int):
            user_id (str):
     """

    accounts: list[AIPersonalAccount]
    cost_micros: int
    name: str
    requests: int
    user_id: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_personal_account import AIPersonalAccount # noqa: PLC0415
        accounts = []
        for accounts_item_data in self.accounts:
            accounts_item = accounts_item_data.to_dict()
            accounts.append(accounts_item)



        cost_micros = self.cost_micros

        name = self.name

        requests = self.requests

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accounts": accounts,
            "costMicros": cost_micros,
            "name": name,
            "requests": requests,
            "userId": user_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_personal_account import AIPersonalAccount # noqa: PLC0415
        d = dict(src_dict)
        accounts = []
        _accounts = d.pop("accounts")
        for accounts_item_data in (_accounts):
            accounts_item = AIPersonalAccount.from_dict(accounts_item_data)



            accounts.append(accounts_item)


        cost_micros = d.pop("costMicros")

        name = d.pop("name")

        requests = d.pop("requests")

        user_id = d.pop("userId")

        ai_personal_person = cls(
            accounts=accounts,
            cost_micros=cost_micros,
            name=name,
            requests=requests,
            user_id=user_id,
        )

        return ai_personal_person

