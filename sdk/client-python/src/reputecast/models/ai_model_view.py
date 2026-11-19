from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_model_view_features_item import AIModelViewFeaturesItem
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.ai_agent_capability import AIAgentCapability
  from ..models.ai_prices_view import AIPricesView





T = TypeVar("T", bound="AIModelView")



@_attrs_define
class AIModelView:
    """ 
        Attributes:
            features (list[AIModelViewFeaturesItem]):
            id (str):
            image_input (bool):
            model (str):
            model_name (str):
            prices (AIPricesView):
            tools (AIAgentCapability):
            deployment (str | Unset):
     """

    features: list[AIModelViewFeaturesItem]
    id: str
    image_input: bool
    model: str
    model_name: str
    prices: AIPricesView
    tools: AIAgentCapability
    deployment: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_agent_capability import AIAgentCapability # noqa: PLC0415
        from ..models.ai_prices_view import AIPricesView # noqa: PLC0415
        features = []
        for features_item_data in self.features:
            features_item = features_item_data.value
            features.append(features_item)



        id = self.id

        image_input = self.image_input

        model = self.model

        model_name = self.model_name

        prices = self.prices.to_dict()

        tools = self.tools.to_dict()

        deployment = self.deployment


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "features": features,
            "id": id,
            "imageInput": image_input,
            "model": model,
            "modelName": model_name,
            "prices": prices,
            "tools": tools,
        })
        if deployment is not UNSET:
            field_dict["deployment"] = deployment

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_agent_capability import AIAgentCapability # noqa: PLC0415
        from ..models.ai_prices_view import AIPricesView # noqa: PLC0415
        d = dict(src_dict)
        features = []
        _features = d.pop("features")
        for features_item_data in (_features):
            features_item = AIModelViewFeaturesItem(features_item_data)



            features.append(features_item)


        id = d.pop("id")

        image_input = d.pop("imageInput")

        model = d.pop("model")

        model_name = d.pop("modelName")

        prices = AIPricesView.from_dict(d.pop("prices"))




        tools = AIAgentCapability.from_dict(d.pop("tools"))




        deployment = d.pop("deployment", UNSET)

        ai_model_view = cls(
            features=features,
            id=id,
            image_input=image_input,
            model=model,
            model_name=model_name,
            prices=prices,
            tools=tools,
            deployment=deployment,
        )

        return ai_model_view

