from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_test_result import SmsTestResult





T = TypeVar("T", bound="SmsTestNumber")



@_attrs_define
class SmsTestNumber:
    """ 
        Attributes:
            color (str):
            country (str):
            display (str):
            id (str):
            name (str):
            phone (str):
            role (str):
            last_test (SmsTestResult | Unset):
            user_id (str | Unset):
     """

    color: str
    country: str
    display: str
    id: str
    name: str
    phone: str
    role: str
    last_test: SmsTestResult | Unset = UNSET
    user_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_test_result import SmsTestResult # noqa: PLC0415
        color = self.color

        country = self.country

        display = self.display

        id = self.id

        name = self.name

        phone = self.phone

        role = self.role

        last_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_test, Unset):
            last_test = self.last_test.to_dict()

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "country": country,
            "display": display,
            "id": id,
            "name": name,
            "phone": phone,
            "role": role,
        })
        if last_test is not UNSET:
            field_dict["lastTest"] = last_test
        if user_id is not UNSET:
            field_dict["userId"] = user_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_test_result import SmsTestResult # noqa: PLC0415
        d = dict(src_dict)
        color = d.pop("color")

        country = d.pop("country")

        display = d.pop("display")

        id = d.pop("id")

        name = d.pop("name")

        phone = d.pop("phone")

        role = d.pop("role")

        _last_test = d.pop("lastTest", UNSET)
        last_test: SmsTestResult | Unset
        if isinstance(_last_test,  Unset):
            last_test = UNSET
        else:
            last_test = SmsTestResult.from_dict(_last_test)




        user_id = d.pop("userId", UNSET)

        sms_test_number = cls(
            color=color,
            country=country,
            display=display,
            id=id,
            name=name,
            phone=phone,
            role=role,
            last_test=last_test,
            user_id=user_id,
        )

        return sms_test_number

