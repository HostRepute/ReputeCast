from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.integration_call_detail_headers import IntegrationCallDetailHeaders
  from ..models.integration_call_view import IntegrationCallView





T = TypeVar("T", bound="IntegrationCallDetail")



@_attrs_define
class IntegrationCallDetail:
    """ 
        Attributes:
            call (IntegrationCallView):
            headers (IntegrationCallDetailHeaders):
            request (Any | Unset):
            response (Any | Unset):
     """

    call: IntegrationCallView
    headers: IntegrationCallDetailHeaders
    request: Any | Unset = UNSET
    response: Any | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_call_detail_headers import IntegrationCallDetailHeaders # noqa: PLC0415
        from ..models.integration_call_view import IntegrationCallView # noqa: PLC0415
        call = self.call.to_dict()

        headers = self.headers.to_dict()

        request = self.request

        response = self.response


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "call": call,
            "headers": headers,
        })
        if request is not UNSET:
            field_dict["request"] = request
        if response is not UNSET:
            field_dict["response"] = response

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_call_detail_headers import IntegrationCallDetailHeaders # noqa: PLC0415
        from ..models.integration_call_view import IntegrationCallView # noqa: PLC0415
        d = dict(src_dict)
        call = IntegrationCallView.from_dict(d.pop("call"))




        headers = IntegrationCallDetailHeaders.from_dict(d.pop("headers"))




        request = d.pop("request", UNSET)

        response = d.pop("response", UNSET)

        integration_call_detail = cls(
            call=call,
            headers=headers,
            request=request,
            response=response,
        )

        return integration_call_detail

